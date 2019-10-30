package template

import (
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/template"
)

func NewEngineWithConfig(tConfig config.TenantConfiguration) *template.Engine {
	// TODO(template): enable file loader only for standalone mode.
	e := template.NewEngine(false)

	e.SetDefault(TemplateItemTypeForgotPasswordEmailTXT, DefaultForgotPasswordEmailTXT)
	e.SetDefault(TemplateItemTypeForgotPasswordResetHTML, DefaultForgotPasswordResetHTML)
	e.SetDefault(TemplateItemTypeForgotPasswordSuccessHTML, DefaultForgotPasswordSuccessHTML)
	e.SetDefault(TemplateItemTypeForgotPasswordErrorHTML, DefaultErrorHTML)
	e.SetDefault(TemplateItemTypeWelcomeEmailTXT, DefaultWelcomeEmailTXT)
	e.SetDefault(TemplateItemTypeUserVerificationSMSTXT, DefaultUserVerificationSMSTXT)
	e.SetDefault(TemplateItemTypeUserVerificationEmailTXT, DefaultUserVerificationEmailTXT)
	e.SetDefault(TemplateItemTypeUserVerificationSuccessHTML, DefaultUserVerificationSuccessHTML)
	e.SetDefault(TemplateItemTypeUserVerificationErrorHTML, DefaultErrorHTML)
	e.SetDefault(TemplateItemTypeMFAOOBCodeSMSTXT, DefaultMFAOOBCodeSMSTXT)
	e.SetDefault(TemplateItemTypeMFAOOBCodeEmailTXT, DefaultMFAOOBCodeEmailTXT)

	return e
}
