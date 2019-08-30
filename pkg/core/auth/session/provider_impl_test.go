package session

import (
	"fmt"
	"net/http"
	"testing"
	gotime "time"

	authtest "github.com/skygeario/skygear-server/pkg/core/auth/testing"

	"github.com/skygeario/skygear-server/pkg/core/config"

	"github.com/skygeario/skygear-server/pkg/core/auth"
	corerand "github.com/skygeario/skygear-server/pkg/core/rand"
	"github.com/skygeario/skygear-server/pkg/core/time"
	. "github.com/smartystreets/goconvey/convey"
)

func TestProvider(t *testing.T) {
	Convey("Provider", t, func() {
		store := NewMockStore()
		eventStore := NewMockEventStore()

		timeProvider := &time.MockProvider{}
		initialTime := gotime.Date(2006, 1, 1, 0, 0, 0, 0, gotime.UTC)
		timeProvider.TimeNow = initialTime
		timeProvider.TimeNowUTC = initialTime

		authContext := authtest.NewMockContext().UseAPIAccessKey("web-app")
		clientConfigs := map[string]config.APIClientConfiguration{
			"web-app":    config.APIClientConfiguration{},
			"mobile-app": config.APIClientConfiguration{},
		}

		req, _ := http.NewRequest("POST", "", nil)
		req.Header.Set("User-Agent", "SDK")
		req.Header.Set("X-Skygear-Extra-Info", `{ "device_name": "Device" }`)
		accessEvent := auth.SessionAccessEvent{
			Timestamp: initialTime,
			UserAgent: "SDK",
			Extra: auth.SessionAccessEventExtraInfo{
				"device_name": "Device",
			},
		}

		var provider Provider = &providerImpl{
			req:           req,
			store:         store,
			eventStore:    eventStore,
			authContext:   authContext,
			clientConfigs: clientConfigs,
			time:          timeProvider,
			rand:          corerand.InsecureRand,
		}

		Convey("creating session", func() {
			Convey("should be successful", func() {
				session, err := provider.Create("user-id", "principal-id")
				So(err, ShouldBeNil)
				So(session, ShouldResemble, &auth.Session{
					ID:                   session.ID,
					ClientID:             "web-app",
					UserID:               "user-id",
					PrincipalID:          "principal-id",
					InitialAccess:        accessEvent,
					CreatedAt:            initialTime,
					AccessedAt:           initialTime,
					AccessToken:          session.AccessToken,
					RefreshToken:         session.RefreshToken,
					AccessTokenCreatedAt: initialTime,
				})
				So(session.AccessToken, ShouldHaveLength, tokenLength+len(session.ID)+1)
				So(eventStore.AccessEvents, ShouldResemble, []auth.SessionAccessEvent{accessEvent})
			})

			Convey("should allow creating multiple sessions for same principal", func() {
				session1, err := provider.Create("user-id", "principal-id")
				So(err, ShouldBeNil)
				So(session1, ShouldResemble, &auth.Session{
					ID:                   session1.ID,
					ClientID:             "web-app",
					UserID:               "user-id",
					PrincipalID:          "principal-id",
					InitialAccess:        accessEvent,
					CreatedAt:            initialTime,
					AccessedAt:           initialTime,
					AccessToken:          session1.AccessToken,
					RefreshToken:         session1.RefreshToken,
					AccessTokenCreatedAt: initialTime,
				})

				session2, err := provider.Create("user-id", "principal-id")
				So(err, ShouldBeNil)
				So(session2, ShouldResemble, &auth.Session{
					ID:                   session2.ID,
					ClientID:             "web-app",
					UserID:               "user-id",
					PrincipalID:          "principal-id",
					InitialAccess:        accessEvent,
					CreatedAt:            initialTime,
					AccessedAt:           initialTime,
					AccessToken:          session2.AccessToken,
					RefreshToken:         session2.RefreshToken,
					AccessTokenCreatedAt: initialTime,
				})

				So(session1.ID, ShouldNotEqual, session2.ID)
			})
			Convey("should generate refresh token if enabled", func() {
				clientConfigs["web-app"] = config.APIClientConfiguration{
					RefreshTokenDisabled: false,
				}
				session, err := provider.Create("user-id", "principal-id")
				So(err, ShouldBeNil)
				So(session.RefreshToken, ShouldHaveLength, tokenLength+len(session.ID)+1)
			})
			Convey("should not generate refresh token if disabled", func() {
				clientConfigs["web-app"] = config.APIClientConfiguration{
					RefreshTokenDisabled: true,
				}
				session, err := provider.Create("user-id", "principal-id")
				So(err, ShouldBeNil)
				So(session.RefreshToken, ShouldBeEmpty)
			})
		})

		Convey("getting session", func() {
			fixtureSession := auth.Session{
				ID:                   "session-id",
				ClientID:             "web-app",
				UserID:               "user-id",
				PrincipalID:          "principal-id",
				CreatedAt:            initialTime,
				AccessedAt:           initialTime,
				AccessToken:          "session-id.access-token",
				RefreshToken:         "session-id.refresh-token",
				AccessTokenCreatedAt: initialTime,
			}
			store.Sessions[fixtureSession.ID] = fixtureSession

			Convey("should be successful using access token", func() {
				session, err := provider.GetByToken("session-id.access-token", auth.SessionTokenKindAccessToken)
				So(err, ShouldBeNil)
				So(session, ShouldResemble, &fixtureSession)
			})

			Convey("should be successful using refresh token", func() {
				session, err := provider.GetByToken("session-id.refresh-token", auth.SessionTokenKindRefreshToken)
				So(err, ShouldBeNil)
				So(session, ShouldResemble, &fixtureSession)
			})

			Convey("should not mix up access & refresh token", func() {
				session, err := provider.GetByToken("session-id.access-token", auth.SessionTokenKindRefreshToken)
				So(err, ShouldBeError, ErrSessionNotFound)
				So(session, ShouldBeNil)

				session, err = provider.GetByToken("session-id.refresh-token", auth.SessionTokenKindAccessToken)
				So(err, ShouldBeError, ErrSessionNotFound)
				So(session, ShouldBeNil)
			})

			Convey("should not match empty tokens", func() {
				Convey("for access token", func() {
					fixtureSession.AccessToken = ""
					store.Sessions[fixtureSession.ID] = fixtureSession

					session, err := provider.GetByToken("session-id.", auth.SessionTokenKindAccessToken)
					So(err, ShouldBeError, ErrSessionNotFound)
					So(session, ShouldBeNil)
				})
				Convey("for refresh token", func() {
					fixtureSession.RefreshToken = ""
					store.Sessions[fixtureSession.ID] = fixtureSession

					session, err := provider.GetByToken("session-id.", auth.SessionTokenKindRefreshToken)
					So(err, ShouldBeError, ErrSessionNotFound)
					So(session, ShouldBeNil)
				})
			})

			Convey("should reject session of other clients", func() {
				authContext.UseAPIAccessKey("mobile-app")
				session, err := provider.GetByToken("session-id.access-token", auth.SessionTokenKindAccessToken)
				So(err, ShouldBeError, ErrSessionNotFound)
				So(session, ShouldBeNil)
			})

			Convey("should reject non-existant session", func() {
				session, err := provider.GetByToken("session-id-unknown.access-token", auth.SessionTokenKindAccessToken)
				So(err, ShouldBeError, ErrSessionNotFound)
				So(session, ShouldBeNil)
			})

			Convey("should reject incorrect token", func() {
				session, err := provider.GetByToken("session-id.incorrect-token", auth.SessionTokenKindAccessToken)
				So(err, ShouldBeError, ErrSessionNotFound)
				So(session, ShouldBeNil)

				session, err = provider.GetByToken("invalid-token", auth.SessionTokenKindAccessToken)
				So(err, ShouldBeError, ErrSessionNotFound)
				So(session, ShouldBeNil)
			})
			Convey("should reject if client is disabled", func() {
				clientConfigs["web-app"] = config.APIClientConfiguration{
					Disabled: true,
				}
				session, err := provider.GetByToken("session-id.access-token", auth.SessionTokenKindAccessToken)
				So(err, ShouldBeError, ErrSessionNotFound)
				So(session, ShouldBeNil)
			})
			Convey("should reject if client does not exists", func() {
				delete(clientConfigs, "web-app")
				session, err := provider.GetByToken("session-id.access-token", auth.SessionTokenKindAccessToken)
				So(err, ShouldBeError, ErrSessionNotFound)
				So(session, ShouldBeNil)
			})
			Convey("should reject if session is expired", func() {
				timeProvider.AdvanceSeconds(1000000)
				session, err := provider.GetByToken("session-id.access-token", auth.SessionTokenKindAccessToken)
				So(err, ShouldBeError, ErrSessionNotFound)
				So(session, ShouldBeNil)
			})
		})

		Convey("accessing session", func() {
			session := auth.Session{
				ID:                   "session-id",
				UserID:               "user-id",
				PrincipalID:          "principal-id",
				CreatedAt:            initialTime,
				AccessedAt:           initialTime,
				AccessToken:          "access-token",
				AccessTokenCreatedAt: initialTime,
			}
			timeProvider.AdvanceSeconds(100)
			timeNow := timeProvider.TimeNowUTC
			accessEvent.Timestamp = timeNow
			store.Sessions["session-id"] = session

			Convey("should be update accessed at time", func() {
				err := provider.Access(&session)
				So(err, ShouldBeNil)
				So(session.AccessedAt, ShouldEqual, timeNow)
			})
			Convey("should be create access event", func() {
				err := provider.Access(&session)
				So(err, ShouldBeNil)
				So(session.LastAccess, ShouldResemble, accessEvent)
				So(eventStore.AccessEvents, ShouldResemble, []auth.SessionAccessEvent{accessEvent})
			})
		})

		Convey("invalidating session", func() {
			store.Sessions["session-id"] = auth.Session{
				ID:                   "session-id",
				UserID:               "user-id",
				PrincipalID:          "principal-id",
				CreatedAt:            initialTime,
				AccessedAt:           initialTime,
				AccessToken:          "access-token",
				AccessTokenCreatedAt: initialTime,
			}

			Convey("should be successful", func() {
				err := provider.Invalidate("session-id")
				So(err, ShouldBeNil)
				So(store.Sessions, ShouldBeEmpty)
			})

			Convey("should be successful for non-existant sessions", func() {
				err := provider.Invalidate("session-id-unknown")
				So(err, ShouldBeNil)
				So(store.Sessions, ShouldNotBeEmpty)
			})
		})

		Convey("listing session", func() {
			makeSession := func(id string, userID string, clientID string, timeOffset int) {
				store.Sessions[id] = auth.Session{
					ID:                   id,
					UserID:               userID,
					ClientID:             clientID,
					CreatedAt:            initialTime.Add(gotime.Duration(timeOffset) * gotime.Second),
					AccessedAt:           initialTime.Add(gotime.Duration(timeOffset) * gotime.Second),
					AccessTokenCreatedAt: initialTime.Add(gotime.Duration(timeOffset) * gotime.Second),
				}
			}
			makeSession("a", "user-1", "web-app", 100)
			makeSession("b", "user-1", "mobile-app", 200)
			makeSession("c", "user-2", "web-app", -10000)
			makeSession("d", "user-2", "disabled-app", 400)
			timeProvider.AdvanceSeconds(500)
			clientConfigs["web-app"] = config.APIClientConfiguration{AccessTokenLifetime: 1000, RefreshTokenDisabled: true}
			clientConfigs["mobile-app"] = config.APIClientConfiguration{AccessTokenLifetime: 1000, RefreshTokenDisabled: true}

			list := func(userID string) (ids []string, err error) {
				sessions, err := provider.List(userID)
				for _, session := range sessions {
					ids = append(ids, session.ID)
				}
				return
			}

			Convey("should be correctly filtered", func() {
				ids, err := list("user-1")
				So(err, ShouldBeNil)
				So(ids, ShouldResemble, []string{"a", "b"})

				ids, err = list("user-2")
				So(err, ShouldBeNil)
				So(ids, ShouldHaveLength, 0)
			})
		})
	})
	Convey("newAccessEvent", t, func() {
		now := gotime.Date(2006, 1, 1, 0, 0, 0, 0, gotime.UTC)
		Convey("should record current timestamp", func() {
			req, _ := http.NewRequest("POST", "", nil)

			event := newAccessEvent(now, req)
			So(event.Timestamp, ShouldResemble, now)
		})
		Convey("should populate connection info", func() {
			req, _ := http.NewRequest("POST", "", nil)
			req.RemoteAddr = "192.168.1.11:31035"
			req.Header.Set("X-Forwarded-For", "13.225.103.28, 216.58.197.110")
			req.Header.Set("X-Real-IP", "216.58.197.110")
			req.Header.Set("Forwarded", "for=216.58.197.110;proto=http;by=192.168.1.11")

			event := newAccessEvent(now, req)
			So(event.Remote, ShouldResemble, auth.SessionAccessEventConnInfo{
				RemoteAddr:    "192.168.1.11:31035",
				XForwardedFor: "13.225.103.28, 216.58.197.110",
				XRealIP:       "216.58.197.110",
				Forwarded:     "for=216.58.197.110;proto=http;by=192.168.1.11",
			})
		})
		Convey("should populate user agent", func() {
			req, _ := http.NewRequest("POST", "", nil)
			req.RemoteAddr = "192.168.1.11:31035"
			req.Header.Set("User-Agent", "SDK")

			event := newAccessEvent(now, req)
			So(event.UserAgent, ShouldEqual, "SDK")
		})
		Convey("should populate extra info", func() {
			req, _ := http.NewRequest("POST", "", nil)
			req.Header.Set("X-Skygear-Extra-Info", `{ "device_name": "Device" }`)

			event := newAccessEvent(now, req)
			So(event.Extra, ShouldResemble, auth.SessionAccessEventExtraInfo{
				"device_name": "Device",
			})
		})
		Convey("should not populate extra info if too large", func() {
			extra := "{ "
			for i := 0; i < 1000; i++ {
				if i != 0 {
					extra += ", "
				}
				extra += fmt.Sprintf(`"info_%d": %d`, i, i)
			}
			extra += " }"

			req, _ := http.NewRequest("POST", "", nil)
			req.Header.Set("X-Skygear-Extra-Info", extra)

			event := newAccessEvent(now, req)
			So(event.Extra, ShouldResemble, auth.SessionAccessEventExtraInfo{})
		})
	})
}
