package config

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/yaml.v2"
)

const inputMinimalYAML = `version: '1'
app_id: 66EAFE32-BF5C-4878-8FC8-DD0EEA440981
app_name: myapp
app_config:
  database_url: postgres://
  database_schema: app
user_config:
  clients: {}
  master_key: masterkey
  auth:
    authentication_session:
      secret: authnsessionsecret
    login_id_keys:
      email:
        type: email
      phone:
        type: phone
      username:
        type: raw
  hook:
    secret: hooksecret
  sso:
    custom_token:
      secret: customtokensecret
    oauth:
      state_jwt_secret: statejwtsecret
`

const inputMinimalJSON = `
{
	"version": "1",
	"app_id": "66EAFE32-BF5C-4878-8FC8-DD0EEA440981",
	"app_name": "myapp",
	"app_config": {
		"database_url": "postgres://",
		"database_schema": "app"
	},
	"user_config": {
		"clients": {},
		"master_key": "masterkey",
		"auth": {
			"authentication_session": {
				"secret": "authnsessionsecret"
			},
			"login_id_keys": {
				"email": {
					"type": "email"
				},
				"phone": {
					"type": "phone"
				},
				"username": {
					"type": "raw"
				}
			},
			"allowed_realms": ["default"]
		},
		"hook": {
			"secret": "hooksecret"
		},
		"sso": {
			"custom_token": {
				"secret": "customtokensecret"
			},
			"oauth": {
				"state_jwt_secret": "statejwtsecret"
			}
		}
	}
}
`

func newInt(i int) *int {
	return &i
}

func makeFullTenantConfig() TenantConfiguration {
	var fullTenantConfig = TenantConfiguration{
		Version: "1",
		AppName: "myapp",
		AppID:   "66EAFE32-BF5C-4878-8FC8-DD0EEA440981",
		AppConfig: AppConfiguration{
			DatabaseURL:    "postgres://user:password@localhost:5432/db?sslmode=disable",
			DatabaseSchema: "app",
			SMTP: SMTPConfiguration{
				Host:     "localhost",
				Port:     465,
				Mode:     "ssl",
				Login:    "user",
				Password: "password",
			},
			Twilio: TwilioConfiguration{
				AccountSID: "mytwilioaccountsid",
				AuthToken:  "mytwilioauthtoken",
				From:       "mytwilio",
			},
			Nexmo: NexmoConfiguration{
				APIKey:    "mynexmoapikey",
				APISecret: "mynexmoapisecret",
				From:      "mynexmo",
			},
			Hook: HookAppConfiguration{
				SyncHookTimeout:      10,
				SyncHookTotalTimeout: 60,
			},
		},
		UserConfig: UserConfiguration{
			Clients: map[string]APIClientConfiguration{
				"web-app": APIClientConfiguration{
					Name:                 "Web App",
					Disabled:             false,
					APIKey:               "api_key",
					SessionTransport:     SessionTransportTypeHeader,
					AccessTokenLifetime:  1800,
					SessionIdleTimeout:   300,
					RefreshTokenLifetime: 86400,
					SameSite:             SessionCookieSameSiteLax,
				},
			},
			MasterKey: "mymasterkey",
			URLPrefix: "http://localhost:3000",
			CORS: CORSConfiguration{
				Origin: "localhost:3000",
			},
			Auth: AuthConfiguration{
				AuthenticationSession: AuthenticationSessionConfiguration{
					Secret: "authnsessionsecret",
				},
				LoginIDKeys: map[string]LoginIDKeyConfiguration{
					"email": LoginIDKeyConfiguration{
						Type:    LoginIDKeyType("email"),
						Minimum: newInt(0),
						Maximum: newInt(1),
					},
					"phone": LoginIDKeyConfiguration{
						Type:    LoginIDKeyType("phone"),
						Minimum: newInt(0),
						Maximum: newInt(1),
					},
					"username": LoginIDKeyConfiguration{
						Type:    LoginIDKeyTypeRaw,
						Minimum: newInt(0),
						Maximum: newInt(1),
					},
				},
				AllowedRealms:              []string{"default"},
				OnUserDuplicateAllowCreate: true,
			},
			MFA: MFAConfiguration{
				Enabled:     true,
				Enforcement: MFAEnforcementOptional,
				Maximum:     newInt(3),
				TOTP: MFATOTPConfiguration{
					Maximum: 1,
				},
				OOB: MFAOOBConfiguration{
					SMS: MFAOOBSMSConfiguration{
						Maximum: 1,
					},
					Email: MFAOOBEmailConfiguration{
						Maximum: 1,
					},
				},
				BearerToken: MFABearerTokenConfiguration{
					ExpireInDays: 60,
				},
				RecoveryCode: MFARecoveryCodeConfiguration{
					Count:       24,
					ListEnabled: true,
				},
			},
			UserAudit: UserAuditConfiguration{
				Enabled:         true,
				TrailHandlerURL: "http://localhost:3000/useraudit",
			},
			PasswordPolicy: PasswordPolicyConfiguration{
				MinLength:             8,
				UppercaseRequired:     true,
				LowercaseRequired:     true,
				DigitRequired:         true,
				SymbolRequired:        true,
				MinimumGuessableLevel: 4,
				ExcludedKeywords:      []string{"admin", "password", "secret"},
				HistorySize:           10,
				HistoryDays:           90,
				ExpiryDays:            30,
			},
			ForgotPassword: ForgotPasswordConfiguration{
				AppName:             "myapp",
				URLPrefix:           "http://localhost:3000/forgotpassword",
				SecureMatch:         true,
				Sender:              "myforgotpasswordsender",
				Subject:             "myforgotpasswordsubject",
				ReplyTo:             "myforgotpasswordreplyto",
				ResetURLLifetime:    60,
				SuccessRedirect:     "http://localhost:3000/forgotpassword/success",
				ErrorRedirect:       "http://localhost:3000/forgotpassword/error",
				EmailTextURL:        "http://localhost:3000/forgotpassword/text",
				EmailHTMLURL:        "http://localhost:3000/forgotpassword/html",
				ResetHTMLURL:        "http://localhost:3000/forgotpassword/reset",
				ResetSuccessHTMLURL: "http://localhost:3000/forgotpassword/reset/success",
				ResetErrorHTMLURL:   "http://localhost:3000/forgotpassword/reset/error",
			},
			WelcomeEmail: WelcomeEmailConfiguration{
				Enabled:     true,
				URLPrefix:   "http://localhost:3000/welcomeemail",
				Sender:      "welcomeemailsender",
				Subject:     "welcomeemailsubject",
				ReplyTo:     "welcomeemailreplyto",
				TextURL:     "http://localhost:3000/welcomeemail/text",
				HTMLURL:     "http://localhost:3000/welcomeemail/html",
				Destination: "first",
			},
			SSO: SSOConfiguration{
				CustomToken: CustomTokenConfiguration{
					Enabled:                    true,
					Issuer:                     "customtokenissuer",
					Audience:                   "customtokenaudience",
					Secret:                     "customtokensecret",
					OnUserDuplicateAllowMerge:  true,
					OnUserDuplicateAllowCreate: true,
				},
				OAuth: OAuthConfiguration{
					URLPrefix:      "http://localhost:3000/oauth",
					StateJWTSecret: "oauthstatejwtsecret",
					AllowedCallbackURLs: []string{
						"http://localhost:3000/oauth/callback",
					},
					ExternalAccessTokenFlowEnabled: true,
					OnUserDuplicateAllowMerge:      true,
					OnUserDuplicateAllowCreate:     true,
					Providers: []OAuthProviderConfiguration{
						OAuthProviderConfiguration{
							ID:           "google",
							Type:         "google",
							ClientID:     "googleclientid",
							ClientSecret: "googleclientsecret",
							Scope:        "email profile",
						},
						OAuthProviderConfiguration{
							ID:           "azure-id-1",
							Type:         "azureadv2",
							ClientID:     "azureclientid",
							ClientSecret: "azureclientsecret",
							Scope:        "email",
							Tenant:       "azure-id-1",
						},
					},
				},
			},
			UserVerification: UserVerificationConfiguration{
				URLPrefix:        "http://localhost:3000/userverification",
				AutoSendOnSignup: true,
				Criteria:         "any",
				ErrorRedirect:    "http://localhost:3000/userverification/error",
				ErrorHTMLURL:     "http://localhost:3000/userverification/error.html",
				LoginIDKeys: map[string]UserVerificationKeyConfiguration{
					"email": UserVerificationKeyConfiguration{

						CodeFormat:      "complex",
						Expiry:          3600,
						SuccessRedirect: "http://localhost:3000/userverification/success",
						SuccessHTMLURL:  "http://localhost:3000/userverification/success.html",
						ErrorRedirect:   "http://localhost:3000/userverification/error",
						ErrorHTMLURL:    "http://localhost:3000/userverification/error.html",
						Provider:        "twilio",
						ProviderConfig: UserVerificationProviderConfiguration{
							Subject: "userverificationsubject",
							Sender:  "userverificationsender",
							ReplyTo: "userverificationreplyto",
							TextURL: "http://localhost:3000/userverification/text",
							HTMLURL: "http://localhost:3000/userverification/html",
						},
					},
				},
			},
			Hook: HookUserConfiguration{
				Secret: "hook-secret",
			},
		},
		Hooks: []Hook{
			Hook{
				Event: "after_signup",
				URL:   "http://localhost:3000/after_signup",
			},
			Hook{
				Event: "after_signup",
				URL:   "http://localhost:3000/after_signup",
			},
		},
		DeploymentRoutes: []DeploymentRoute{
			DeploymentRoute{
				Version: "a",
				Path:    "/",
				Type:    "http-service",
				TypeConfig: map[string]interface{}{
					"backend_url": "http://localhost:3000",
				},
			},
			DeploymentRoute{
				Version: "a",
				Path:    "/api",
				Type:    "http-service",
				TypeConfig: map[string]interface{}{
					"backend_url": "http://localhost:3001",
				},
			},
		},
	}

	return fullTenantConfig
}

func TestTenantConfig(t *testing.T) {
	Convey("Test TenantConfiguration", t, func() {
		// YAML
		Convey("should load tenant config from YAML", func() {
			c, err := NewTenantConfigurationFromYAML(strings.NewReader(inputMinimalYAML))
			So(err, ShouldBeNil)

			So(c.Version, ShouldEqual, "1")
			So(c.AppName, ShouldEqual, "myapp")
			So(c.AppConfig.DatabaseURL, ShouldEqual, "postgres://")
			So(c.UserConfig.Clients, ShouldBeEmpty)
			So(c.UserConfig.MasterKey, ShouldEqual, "masterkey")
		})
		Convey("should have default value when load from YAML", func() {
			c, err := NewTenantConfigurationFromYAML(strings.NewReader(inputMinimalYAML))
			So(err, ShouldBeNil)
			So(c.UserConfig.CORS.Origin, ShouldEqual, "*")
			So(c.AppConfig.SMTP.Port, ShouldEqual, 25)
		})
		Convey("should validate when load from YAML", func() {
			invalidInput := `
app_id: 66EAFE32-BF5C-4878-8FC8-DD0EEA440981
app_name: myapp
app_config:
  database_url: postgres://
  database_schema: app
user_config:
  clients: {}
  master_key: masterkey
`
			_, err := NewTenantConfigurationFromYAML(strings.NewReader(invalidInput))
			So(err, ShouldBeError, "Only version 1 is supported")
		})
		// JSON
		Convey("should have default value when load from JSON", func() {
			c, err := NewTenantConfigurationFromJSON(strings.NewReader(inputMinimalJSON), false)
			So(err, ShouldBeNil)
			So(c.Version, ShouldEqual, "1")
			So(c.AppName, ShouldEqual, "myapp")
			So(c.AppConfig.DatabaseURL, ShouldEqual, "postgres://")
			So(c.UserConfig.Clients, ShouldBeEmpty)
			So(c.UserConfig.MasterKey, ShouldEqual, "masterkey")
			So(c.UserConfig.CORS.Origin, ShouldEqual, "*")
			So(c.AppConfig.SMTP.Port, ShouldEqual, 25)
		})
		Convey("should validate when load from JSON", func() {
			invalidInput := `
		{
		  "app_id": "66EAFE32-BF5C-4878-8FC8-DD0EEA440981",
		  "app_name": "myapp",
		  "app_config": {
		    "database_url": "postgres://",
		    "database_schema": "app"
		  },
		  "user_config": {
		    "api_key": "apikey",
		    "master_key": "masterkey",
		    "welcome_email": {
		      "enabled": true
		    }
		  }
		}
					`
			_, err := NewTenantConfigurationFromJSON(strings.NewReader(invalidInput), false)
			So(err, ShouldBeError, "Only version 1 is supported")
		})
		// Conversion
		Convey("should be losslessly converted between Go and msgpack", func() {
			c := makeFullTenantConfig()
			base64msgpack, err := c.StdBase64Msgpack()
			So(err, ShouldBeNil)
			cc, err := NewTenantConfigurationFromStdBase64Msgpack(base64msgpack)
			So(err, ShouldBeNil)
			So(c, ShouldResemble, *cc)
		})
		Convey("should be losslessly converted between Go and JSON", func() {
			c := makeFullTenantConfig()
			b, err := json.Marshal(c)
			So(err, ShouldBeNil)
			cc, err := NewTenantConfigurationFromJSON(bytes.NewReader(b), false)
			So(err, ShouldBeNil)
			So(c, ShouldResemble, *cc)
		})
		Convey("should be losslessly converted between Go and YAML", func() {
			c := makeFullTenantConfig()
			b, err := yaml.Marshal(c)
			So(err, ShouldBeNil)
			cc, err := NewTenantConfigurationFromYAML(bytes.NewReader(b))
			So(err, ShouldBeNil)
			So(c, ShouldResemble, *cc)
		})
		Convey("should set OAuth provider id and default scope", func() {
			c := makeFullTenantConfig()
			c.UserConfig.SSO.OAuth.Providers = []OAuthProviderConfiguration{
				OAuthProviderConfiguration{
					Type:         OAuthProviderTypeGoogle,
					ClientID:     "googleclientid",
					ClientSecret: "googleclientsecret",
				},
			}
			c.AfterUnmarshal()

			google := c.UserConfig.SSO.OAuth.Providers[0]

			So(google.ID, ShouldEqual, OAuthProviderTypeGoogle)
			So(google.Scope, ShouldEqual, "profile email")
		})
		Convey("should validate api key != master key", func() {
			c := makeFullTenantConfig()
			clientConfig := c.UserConfig.Clients["web-app"]
			clientConfig.APIKey = c.UserConfig.MasterKey
			c.UserConfig.Clients["web-app"] = clientConfig

			err := c.Validate()
			So(err, ShouldBeError, "Master key must not be same as API key")
		})
		Convey("should validate minimum <= maximum", func() {
			c := makeFullTenantConfig()
			email := c.UserConfig.Auth.LoginIDKeys["email"]
			email.Minimum = newInt(2)
			email.Maximum = newInt(1)
			c.UserConfig.Auth.LoginIDKeys["email"] = email
			err := c.Validate()
			So(err, ShouldBeError, "Invalid LoginIDKeys amount range: email")
		})
		Convey("UserVerification.LoginIDKeys is subset of Auth.LoginIDKeys", func() {
			c := makeFullTenantConfig()
			invalid := c.UserConfig.UserVerification.LoginIDKeys["email"]
			c.UserConfig.UserVerification.LoginIDKeys["invalid"] = invalid
			err := c.Validate()
			So(err, ShouldBeError, "Cannot verify disallowed login ID key: invalid")
		})
		Convey("should validate OAuth Provider", func() {
			c := makeFullTenantConfig()
			c.UserConfig.SSO.OAuth.Providers = []OAuthProviderConfiguration{
				OAuthProviderConfiguration{
					ID:           "azure",
					Type:         OAuthProviderTypeAzureADv2,
					ClientID:     "clientid",
					ClientSecret: "clientsecret",
					Tenant:       "tenant",
				},
				OAuthProviderConfiguration{
					ID:           "azure",
					Type:         OAuthProviderTypeAzureADv2,
					ClientID:     "clientid",
					ClientSecret: "clientsecret",
					Tenant:       "tenant",
				},
			}

			So(c.Validate(), ShouldBeError, "Duplicate OAuth Provider: azure")
		})
	})
}