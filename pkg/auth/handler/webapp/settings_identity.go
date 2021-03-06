package webapp

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/skygeario/skygear-server/pkg/auth"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/webapp"
	"github.com/skygeario/skygear-server/pkg/core/db"
)

func AttachSettingsIdentityHandler(
	router *mux.Router,
	authDependency auth.DependencyMap,
) {
	router.
		NewRoute().
		Path("/settings/identity").
		Handler(auth.MakeHandler(authDependency, newSettingsIdentityHandler))
}

type settingsIdentityProvider interface {
	GetSettingsIdentity(w http.ResponseWriter, r *http.Request) (func(error), error)
	LinkIdentityProvider(w http.ResponseWriter, r *http.Request, providerAlias string) (func(error), error)
	UnlinkIdentityProvider(w http.ResponseWriter, r *http.Request, providerAlias string) (func(error), error)
	AddOrChangeLoginID(w http.ResponseWriter, r *http.Request) (func(error), error)
}

type SettingsIdentityHandler struct {
	RenderProvider webapp.RenderProvider
	Provider       settingsIdentityProvider
	TxContext      db.TxContext
}

func (h *SettingsIdentityHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	db.WithTx(h.TxContext, func() error {
		if r.Method == "GET" {
			writeResponse, err := h.Provider.GetSettingsIdentity(w, r)
			writeResponse(err)
			return err
		}

		if r.Method == "POST" {
			if r.Form.Get("x_action") == "link" {
				writeResponse, err := h.Provider.LinkIdentityProvider(w, r, r.Form.Get("x_idp_id"))
				writeResponse(err)
				return err
			}
			if r.Form.Get("x_action") == "unlink" {
				writeResponse, err := h.Provider.UnlinkIdentityProvider(w, r, r.Form.Get("x_idp_id"))
				writeResponse(err)
				return err
			}
			if r.Form.Get("x_action") == "login_id" {
				writeResponse, err := h.Provider.AddOrChangeLoginID(w, r)
				writeResponse(err)
				return err
			}
		}

		return nil
	})
}
