package config

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/skygeario/skygear-server/pkg/core/auth/metadata"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"

	coreHttp "github.com/skygeario/skygear-server/pkg/core/http"
	"github.com/skygeario/skygear-server/pkg/core/name"
	"github.com/skygeario/skygear-server/pkg/core/validation"
)

//go:generate msgp -tests=false
type TenantConfiguration struct {
	Version          string            `json:"version,omitempty" yaml:"version" msg:"version"`
	AppName          string            `json:"app_name,omitempty" yaml:"app_name" msg:"app_name"`
	AppConfig        AppConfiguration  `json:"app_config,omitempty" yaml:"app_config" msg:"app_config"`
	UserConfig       UserConfiguration `json:"user_config,omitempty" yaml:"user_config" msg:"user_config"`
	Hooks            []Hook            `json:"hooks,omitempty" yaml:"hooks" msg:"hooks"`
	DeploymentRoutes []DeploymentRoute `json:"deployment_routes,omitempty" yaml:"deployment_routes" msg:"deployment_routes"`
}

type Hook struct {
	Event string `json:"event,omitempty" yaml:"event" msg:"event"`
	URL   string `json:"url,omitempty" yaml:"url" msg:"url"`
}

type DeploymentRoute struct {
	Version    string                 `json:"version,omitempty" yaml:"version" msg:"version"`
	Path       string                 `json:"path,omitempty" yaml:"path" msg:"path"`
	Type       string                 `json:"type,omitempty" yaml:"type" msg:"type"`
	TypeConfig map[string]interface{} `json:"type_config,omitempty" yaml:"type_config" msg:"type_config"`
}

func defaultAppConfiguration() AppConfiguration {
	return AppConfiguration{
		DatabaseURL: "postgres://postgres:@localhost/postgres?sslmode=disable",
		SMTP: SMTPConfiguration{
			Port: 25,
			Mode: "normal",
		},
	}
}

func defaultUserConfiguration() UserConfiguration {
	return UserConfiguration{
		CORS: CORSConfiguration{
			Origin: "*",
		},
		Auth: AuthConfiguration{
			// Default to email and username
			LoginIDKeys: map[string]LoginIDKeyConfiguration{
				"username": LoginIDKeyConfiguration{Type: LoginIDKeyTypeRaw},
				"email":    LoginIDKeyConfiguration{Type: LoginIDKeyType(metadata.Email)},
				"phone":    LoginIDKeyConfiguration{Type: LoginIDKeyType(metadata.Phone)},
			},
			AllowedRealms: []string{"default"},
		},
		ForgotPassword: ForgotPasswordConfiguration{
			SecureMatch:      false,
			Sender:           "no-reply@skygeario.com",
			Subject:          "Reset password instruction",
			ResetURLLifetime: 43200,
		},
		WelcomeEmail: WelcomeEmailConfiguration{
			Enabled:     false,
			Sender:      "no-reply@skygeario.com",
			Subject:     "Welcome!",
			Destination: WelcomeEmailDestinationFirst,
		},
	}
}

type FromScratchOptions struct {
	AppName     string `envconfig:"APP_NAME"`
	DatabaseURL string `envconfig:"DATABASE_URL"`
	APIKey      string `envconfig:"API_KEY"`
	MasterKey   string `envconfig:"MASTER_KEY"`
}

func NewTenantConfigurationFromScratch(options FromScratchOptions) (*TenantConfiguration, error) {
	c := TenantConfiguration{
		AppConfig:  defaultAppConfiguration(),
		UserConfig: defaultUserConfiguration(),
	}
	c.Version = "1"

	c.AppName = options.AppName
	c.AppConfig.DatabaseURL = options.DatabaseURL
	c.UserConfig.APIKey = options.APIKey
	c.UserConfig.MasterKey = options.MasterKey

	c.AfterUnmarshal()
	err := c.Validate()
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func loadTenantConfigurationFromYAML(r io.Reader) (*TenantConfiguration, error) {
	decoder := yaml.NewDecoder(r)
	config := TenantConfiguration{
		AppConfig:  defaultAppConfiguration(),
		UserConfig: defaultUserConfiguration(),
	}
	err := decoder.Decode(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func NewTenantConfigurationFromYAML(r io.Reader) (*TenantConfiguration, error) {
	config, err := loadTenantConfigurationFromYAML(r)
	if err != nil {
		return nil, err
	}

	config.AfterUnmarshal()
	err = config.Validate()
	if err != nil {
		return nil, err
	}
	return config, nil
}

func NewTenantConfigurationFromEnv() (*TenantConfiguration, error) {
	options := FromScratchOptions{}
	err := envconfig.Process("", &options)
	if err != nil {
		return nil, err
	}
	return NewTenantConfigurationFromScratch(options)
}

func NewTenantConfigurationFromYAMLAndEnv(open func() (io.Reader, error)) (*TenantConfiguration, error) {
	options := FromScratchOptions{}
	err := envconfig.Process("", &options)
	if err != nil {
		return nil, err
	}

	r, err := open()
	if err != nil {
		// Load from env directly
		return NewTenantConfigurationFromScratch(options)
	}
	defer func() {
		if rc, ok := r.(io.Closer); ok {
			rc.Close()
		}
	}()

	c, err := loadTenantConfigurationFromYAML(r)
	if err != nil {
		return nil, err
	}

	// Allow override from env
	if options.AppName != "" {
		c.AppName = options.AppName
	}
	if options.DatabaseURL != "" {
		c.AppConfig.DatabaseURL = options.DatabaseURL
	}
	if options.APIKey != "" {
		c.UserConfig.APIKey = options.APIKey
	}
	if options.MasterKey != "" {
		c.UserConfig.MasterKey = options.MasterKey
	}

	c.AfterUnmarshal()
	err = c.Validate()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func NewTenantConfigurationFromJSON(r io.Reader) (*TenantConfiguration, error) {
	decoder := json.NewDecoder(r)
	config := TenantConfiguration{
		AppConfig:  defaultAppConfiguration(),
		UserConfig: defaultUserConfiguration(),
	}
	err := decoder.Decode(&config)
	if err != nil {
		return nil, err
	}
	config.AfterUnmarshal()
	err = config.Validate()
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func NewTenantConfigurationFromStdBase64Msgpack(s string) (*TenantConfiguration, error) {
	bytes, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	var config TenantConfiguration
	_, err = config.UnmarshalMsg(bytes)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *TenantConfiguration) Value() (driver.Value, error) {
	bytes, err := json.Marshal(*c)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (c *TenantConfiguration) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("Cannot convert %T to TenantConfiguration", value)
	}
	config, err := NewTenantConfigurationFromJSON(bytes.NewReader(b))
	if err != nil {
		return err
	}
	*c = *config
	return nil
}

func (c *TenantConfiguration) StdBase64Msgpack() (string, error) {
	bytes, err := c.MarshalMsg(nil)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

func (c *TenantConfiguration) GetOAuthProviderByID(id string) (OAuthProviderConfiguration, bool) {
	for _, provider := range c.UserConfig.SSO.OAuth.Providers {
		if provider.ID == id {
			return provider, true
		}
	}
	return OAuthProviderConfiguration{}, false
}

func (c *TenantConfiguration) DefaultSensitiveLoggerValues() []string {
	return []string{
		c.UserConfig.APIKey,
		c.UserConfig.MasterKey,
	}
}

// nolint: gocyclo
func (c *TenantConfiguration) Validate() error {
	if c.Version != "1" {
		return errors.New("Only version 1 is supported")
	}

	// Validate AppConfiguration
	if c.AppConfig.DatabaseURL == "" {
		return errors.New("DATABASE_URL is not set")
	}
	if !c.AppConfig.SMTP.Mode.IsValid() {
		return errors.New("Invalid SMTP mode")
	}

	// Validate AppName
	if c.AppName == "" {
		return errors.New("APP_NAME is not set")
	}
	if err := name.ValidateAppName(c.AppName); err != nil {
		return err
	}

	// Validate UserConfiguration
	if err := validation.ValidateUserConfiguration(c.UserConfig); err != nil {
		return err
	}

	// Validate complex UserConfiguration
	if c.UserConfig.APIKey == c.UserConfig.MasterKey {
		return errors.New("MASTER_KEY cannot be the same as API_KEY")
	}

	for _, loginIDKeyConfig := range c.UserConfig.Auth.LoginIDKeys {
		if *loginIDKeyConfig.Minimum > *loginIDKeyConfig.Maximum || *loginIDKeyConfig.Maximum <= 0 {
			return errors.New("Invalid LoginIDKeys amount range: " + string(loginIDKeyConfig.Type))
		}
	}

	for key := range c.UserConfig.UserVerification.LoginIDKeys {
		_, ok := c.UserConfig.Auth.LoginIDKeys[key]
		if !ok {
			return errors.New("Cannot verify disallowed login ID key: " + key)
		}
	}

	// Validate OAuth
	seenOAuthProviderID := map[string]struct{}{}
	for _, provider := range c.UserConfig.SSO.OAuth.Providers {
		// Ensure ID is not duplicate.
		if _, ok := seenOAuthProviderID[provider.ID]; ok {
			return fmt.Errorf("Duplicate OAuth Provider: %s", provider.ID)
		}
		seenOAuthProviderID[provider.ID] = struct{}{}
	}

	return nil
}

// nolint: gocyclo
func (c *TenantConfiguration) AfterUnmarshal() {
	// Default token secret to master key
	if c.UserConfig.TokenStore.Secret == "" {
		c.UserConfig.TokenStore.Secret = c.UserConfig.MasterKey
	}
	// Default oauth state secret to master key
	if c.UserConfig.SSO.OAuth.StateJWTSecret == "" {
		c.UserConfig.SSO.OAuth.StateJWTSecret = c.UserConfig.MasterKey
	}

	// Propagate AppName
	if c.UserConfig.ForgotPassword.AppName == "" {
		c.UserConfig.ForgotPassword.AppName = c.AppName
	}

	// Propagate URLPrefix
	if c.UserConfig.ForgotPassword.URLPrefix == "" {
		c.UserConfig.ForgotPassword.URLPrefix = c.UserConfig.URLPrefix
	}
	if c.UserConfig.WelcomeEmail.URLPrefix == "" {
		c.UserConfig.WelcomeEmail.URLPrefix = c.UserConfig.URLPrefix
	}
	if c.UserConfig.SSO.OAuth.URLPrefix == "" {
		c.UserConfig.SSO.OAuth.URLPrefix = c.UserConfig.URLPrefix
	}
	if c.UserConfig.UserVerification.URLPrefix == "" {
		c.UserConfig.UserVerification.URLPrefix = c.UserConfig.URLPrefix
	}

	// Remove trailing slash in URLs
	c.UserConfig.URLPrefix = removeTrailingSlash(c.UserConfig.URLPrefix)
	c.UserConfig.ForgotPassword.URLPrefix = removeTrailingSlash(c.UserConfig.ForgotPassword.URLPrefix)
	c.UserConfig.WelcomeEmail.URLPrefix = removeTrailingSlash(c.UserConfig.WelcomeEmail.URLPrefix)
	c.UserConfig.UserVerification.URLPrefix = removeTrailingSlash(c.UserConfig.UserVerification.URLPrefix)
	c.UserConfig.SSO.OAuth.URLPrefix = removeTrailingSlash(c.UserConfig.SSO.OAuth.URLPrefix)

	// Set default value for login ID keys config
	for key, config := range c.UserConfig.Auth.LoginIDKeys {
		if config.Minimum == nil {
			config.Minimum = new(int)
			*config.Minimum = 0
		}
		if config.Maximum == nil {
			config.Maximum = new(int)
			if *config.Minimum == 0 {
				*config.Maximum = 1
			} else {
				*config.Maximum = *config.Minimum
			}
		}
		c.UserConfig.Auth.LoginIDKeys[key] = config
	}

	// Set default user verification settings
	if c.UserConfig.UserVerification.Criteria == "" {
		c.UserConfig.UserVerification.Criteria = UserVerificationCriteriaAny
	}
	for key, config := range c.UserConfig.UserVerification.LoginIDKeys {
		if config.CodeFormat == "" {
			config.CodeFormat = UserVerificationCodeFormatComplex
		}
		if config.Expiry == 0 {
			config.Expiry = 3600 // 1 hour
		}
		if config.ProviderConfig.Sender == "" {
			config.ProviderConfig.Sender = "no-reply@skygeario.com"
		}
		if config.ProviderConfig.Subject == "" {
			config.ProviderConfig.Subject = "Verification instruction"
		}
		c.UserConfig.UserVerification.LoginIDKeys[key] = config
	}

	// Set default welcome email destination
	if c.UserConfig.WelcomeEmail.Destination == "" {
		c.UserConfig.WelcomeEmail.Destination = WelcomeEmailDestinationFirst
	}

	// Set default smtp mode
	if c.AppConfig.SMTP.Mode == "" {
		c.AppConfig.SMTP.Mode = SMTPModeNormal
	}

	// Set type to id
	// Set default scope for OAuth Provider
	for i, provider := range c.UserConfig.SSO.OAuth.Providers {
		if provider.ID == "" {
			c.UserConfig.SSO.OAuth.Providers[i].ID = string(provider.Type)
		}
		switch provider.Type {
		case OAuthProviderTypeGoogle:
			if provider.Scope == "" {
				// https://developers.google.com/identity/protocols/googlescopes#google_sign-in
				c.UserConfig.SSO.OAuth.Providers[i].Scope = "profile email"
			}
		case OAuthProviderTypeFacebook:
			if provider.Scope == "" {
				// https://developers.facebook.com/docs/facebook-login/permissions/#reference-default
				// https://developers.facebook.com/docs/facebook-login/permissions/#reference-email
				c.UserConfig.SSO.OAuth.Providers[i].Scope = "default email"
			}
		case OAuthProviderTypeInstagram:
			if provider.Scope == "" {
				// https://www.instagram.com/developer/authorization/
				c.UserConfig.SSO.OAuth.Providers[i].Scope = "basic"
			}
		case OAuthProviderTypeLinkedIn:
			if provider.Scope == "" {
				// https://docs.microsoft.com/en-us/linkedin/shared/integrations/people/profile-api?context=linkedin/compliance/context
				// https://docs.microsoft.com/en-us/linkedin/shared/integrations/people/primary-contact-api?context=linkedin/compliance/context
				c.UserConfig.SSO.OAuth.Providers[i].Scope = "r_liteprofile r_emailaddress"
			}
		case OAuthProviderTypeAzureADv2:
			if provider.Scope == "" {
				// https://docs.microsoft.com/en-us/azure/active-directory/develop/v2-permissions-and-consent#openid-connect-scopes
				c.UserConfig.SSO.OAuth.Providers[i].Scope = "openid profile email"
			}
		}
	}

	// Set default hook secret
	if c.UserConfig.Hook.Secret == "" {
		c.UserConfig.Hook.Secret = c.UserConfig.MasterKey
	}

	// Set default hook timeout
	if c.AppConfig.Hook.SyncHookTimeout == 0 {
		c.AppConfig.Hook.SyncHookTimeout = 5
	}
	if c.AppConfig.Hook.SyncHookTotalTimeout == 0 {
		c.AppConfig.Hook.SyncHookTotalTimeout = 10
	}
}

func GetTenantConfig(r *http.Request) TenantConfiguration {
	s := r.Header.Get(coreHttp.HeaderTenantConfig)
	config, err := NewTenantConfigurationFromStdBase64Msgpack(s)
	if err != nil {
		panic(err)
	}
	return *config
}

func SetTenantConfig(r *http.Request, config *TenantConfiguration) {
	value, err := config.StdBase64Msgpack()
	if err != nil {
		panic(err)
	}
	r.Header.Set(coreHttp.HeaderTenantConfig, value)
}

func DelTenantConfig(r *http.Request) {
	r.Header.Del(coreHttp.HeaderTenantConfig)
}

func removeTrailingSlash(url string) string {
	if strings.HasSuffix(url, "/") {
		return url[:len(url)-1]
	}

	return url
}

// UserConfiguration represents user-editable configuration
type UserConfiguration struct {
	APIKey           string                        `json:"api_key,omitempty" yaml:"api_key" msg:"api_key"`
	MasterKey        string                        `json:"master_key,omitempty" yaml:"master_key" msg:"master_key"`
	URLPrefix        string                        `json:"url_prefix,omitempty" yaml:"url_prefix" msg:"url_prefix"`
	CORS             CORSConfiguration             `json:"cors,omitempty" yaml:"cors" msg:"cors"`
	Auth             AuthConfiguration             `json:"auth,omitempty" yaml:"auth" msg:"auth"`
	TokenStore       TokenStoreConfiguration       `json:"token_store,omitempty" yaml:"token_store" msg:"token_store"`
	UserAudit        UserAuditConfiguration        `json:"user_audit,omitempty" yaml:"user_audit" msg:"user_audit"`
	ForgotPassword   ForgotPasswordConfiguration   `json:"forgot_password,omitempty" yaml:"forgot_password" msg:"forgot_password"`
	WelcomeEmail     WelcomeEmailConfiguration     `json:"welcome_email,omitempty" yaml:"welcome_email" msg:"welcome_email"`
	SSO              SSOConfiguration              `json:"sso,omitempty" yaml:"sso" msg:"sso"`
	UserVerification UserVerificationConfiguration `json:"user_verification,omitempty" yaml:"user_verification" msg:"user_verification"`
	Hook             HookUserConfiguration         `json:"hook,omitempty" yaml:"hook" msg:"hook"`
}

// CORSConfiguration represents CORS configuration.
// Currently we only support configuring origin.
// We may allow to support other headers in the future.
// The interpretation of origin is done by this library
// https://github.com/iawaknahc/originmatcher
type CORSConfiguration struct {
	Origin string `json:"origin,omitempty" yaml:"origin" msg:"origin"`
}

type AuthConfiguration struct {
	LoginIDKeys                map[string]LoginIDKeyConfiguration `json:"login_id_keys,omitempty" yaml:"login_id_keys" msg:"login_id_keys"`
	AllowedRealms              []string                           `json:"allowed_realms,omitempty" yaml:"allowed_realms" msg:"allowed_realms"`
	OnUserDuplicateAllowCreate bool                               `json:"on_user_duplicate_allow_create,omitempty" yaml:"on_user_duplicate_allow_create" msg:"on_user_duplicate_allow_create"`
}

type LoginIDKeyType string

const LoginIDKeyTypeRaw LoginIDKeyType = "raw"

func (t LoginIDKeyType) MetadataKey() (metadata.StandardKey, bool) {
	for _, key := range metadata.AllKeys() {
		if string(t) == string(key) {
			return key, true
		}
	}
	return "", false
}

func (t LoginIDKeyType) IsValid() bool {
	_, validKey := t.MetadataKey()
	return t == LoginIDKeyTypeRaw || validKey
}

type LoginIDKeyConfiguration struct {
	Type    LoginIDKeyType `json:"type,omitempty" yaml:"type" msg:"type"`
	Minimum *int           `json:"minimum,omitempty" yaml:"minimum" msg:"minimum"`
	Maximum *int           `json:"maximum,omitempty" yaml:"maximum" msg:"maximum"`
}

type TokenStoreConfiguration struct {
	Secret string `json:"secret,omitempty" yaml:"secret" msg:"secret"`
	Expiry int64  `json:"expiry,omitempty" yaml:"expiry" msg:"expiry"`
}

type UserAuditConfiguration struct {
	Enabled         bool                  `json:"enabled,omitempty" yaml:"enabled" msg:"enabled"`
	TrailHandlerURL string                `json:"trail_handler_url,omitempty" yaml:"trail_handler_url" msg:"trail_handler_url"`
	Password        PasswordConfiguration `json:"password,omitempty" yaml:"password" msg:"password"`
}

type PasswordConfiguration struct {
	MinLength             int      `json:"min_length,omitempty" yaml:"min_length" msg:"min_length"`
	UppercaseRequired     bool     `json:"uppercase_required,omitempty" yaml:"uppercase_required" msg:"uppercase_required"`
	LowercaseRequired     bool     `json:"lowercase_required,omitempty" yaml:"lowercase_required" msg:"lowercase_required"`
	DigitRequired         bool     `json:"digit_required,omitempty" yaml:"digit_required" msg:"digit_required"`
	SymbolRequired        bool     `json:"symbol_required,omitempty" yaml:"symbol_required" msg:"symbol_required"`
	MinimumGuessableLevel int      `json:"minimum_guessable_level,omitempty" yaml:"minimum_guessable_level" msg:"minimum_guessable_level"`
	ExcludedKeywords      []string `json:"excluded_keywords,omitempty" yaml:"excluded_keywords" msg:"excluded_keywords"`
	// Do not know how to support fields because we do not
	// have them now
	// ExcludedFields     []string `json:"excluded_fields,omitempty" yaml:"excluded_fields" msg:"excluded_fields"`
	HistorySize int `json:"history_size,omitempty" yaml:"history_size" msg:"history_size"`
	HistoryDays int `json:"history_days,omitempty" yaml:"history_days" msg:"history_days"`
	ExpiryDays  int `json:"expiry_days,omitempty" yaml:"expiry_days" msg:"expiry_days"`
}

type ForgotPasswordConfiguration struct {
	AppName             string `json:"app_name,omitempty" yaml:"app_name" msg:"app_name"`
	URLPrefix           string `json:"url_prefix,omitempty" yaml:"url_prefix" msg:"url_prefix"`
	SecureMatch         bool   `json:"secure_match,omitempty" yaml:"secure_match" msg:"secure_match"`
	SenderName          string `json:"sender_name,omitempty" yaml:"sender_name" msg:"sender_name"`
	Sender              string `json:"sender,omitempty" yaml:"sender" msg:"sender"`
	Subject             string `json:"subject,omitempty" yaml:"subject" msg:"subject"`
	ReplyToName         string `json:"reply_to_name,omitempty" yaml:"reply_to_name" msg:"reply_to_name"`
	ReplyTo             string `json:"reply_to,omitempty" yaml:"reply_to" msg:"reply_to"`
	ResetURLLifetime    int    `json:"reset_url_lifetime,omitempty" yaml:"reset_url_lifetime" msg:"reset_url_lifetime"`
	SuccessRedirect     string `json:"success_redirect,omitempty" yaml:"success_redirect" msg:"success_redirect"`
	ErrorRedirect       string `json:"error_redirect,omitempty" yaml:"error_redirect" msg:"error_redirect"`
	EmailTextURL        string `json:"email_text_url,omitempty" yaml:"email_text_url" msg:"email_text_url"`
	EmailHTMLURL        string `json:"email_html_url,omitempty" yaml:"email_html_url" msg:"email_html_url"`
	ResetHTMLURL        string `json:"reset_html_url,omitempty" yaml:"reset_html_url" msg:"reset_html_url"`
	ResetSuccessHTMLURL string `json:"reset_success_html_url,omitempty" yaml:"reset_success_html_url" msg:"reset_success_html_url"`
	ResetErrorHTMLURL   string `json:"reset_error_html_url,omitempty" yaml:"reset_error_html_url" msg:"reset_error_html_url"`
}

type WelcomeEmailDestination string

const (
	WelcomeEmailDestinationFirst WelcomeEmailDestination = "first"
	WelcomeEmailDestinationAll   WelcomeEmailDestination = "all"
)

func (destination WelcomeEmailDestination) IsValid() bool {
	return destination == WelcomeEmailDestinationFirst || destination == WelcomeEmailDestinationAll
}

type WelcomeEmailConfiguration struct {
	Enabled     bool                    `json:"enabled,omitempty" yaml:"enabled" msg:"enabled"`
	URLPrefix   string                  `json:"url_prefix,omitempty" yaml:"url_prefix" msg:"url_prefix"`
	SenderName  string                  `json:"sender_name,omitempty" yaml:"sender_name" msg:"sender_name"`
	Sender      string                  `json:"sender,omitempty" yaml:"sender" msg:"sender"`
	Subject     string                  `json:"subject,omitempty" yaml:"subject" msg:"subject"`
	ReplyToName string                  `json:"reply_to_name,omitempty" yaml:"reply_to_name" msg:"reply_to_name"`
	ReplyTo     string                  `json:"reply_to,omitempty" yaml:"reply_to" msg:"reply_to"`
	TextURL     string                  `json:"text_url,omitempty" yaml:"text_url" msg:"text_url"`
	HTMLURL     string                  `json:"html_url,omitempty" yaml:"html_url" msg:"html_url"`
	Destination WelcomeEmailDestination `json:"destination,omitempty" yaml:"destination" msg:"destination"`
}

type SSOConfiguration struct {
	CustomToken CustomTokenConfiguration `json:"custom_token,omitempty" yaml:"custom_token" msg:"custom_token"`
	OAuth       OAuthConfiguration       `json:"oauth,omitempty" yaml:"oauth" msg:"oauth"`
}

type CustomTokenConfiguration struct {
	Enabled                    bool   `json:"enabled,omitempty" yaml:"enabled" msg:"enabled"`
	Issuer                     string `json:"issuer,omitempty" yaml:"issuer" msg:"issuer"`
	Audience                   string `json:"audience,omitempty" yaml:"audience" msg:"audience"`
	Secret                     string `json:"secret,omitempty" yaml:"secret" msg:"secret"`
	OnUserDuplicateAllowMerge  bool   `json:"on_user_duplicate_allow_merge,omitempty" yaml:"on_user_duplicate_allow_merge" msg:"on_user_duplicate_allow_merge"`
	OnUserDuplicateAllowCreate bool   `json:"on_user_duplicate_allow_create,omitempty" yaml:"on_user_duplicate_allow_create" msg:"on_user_duplicate_allow_create"`
}

type OAuthConfiguration struct {
	URLPrefix                      string                       `json:"url_prefix,omitempty" yaml:"url_prefix" msg:"url_prefix"`
	StateJWTSecret                 string                       `json:"state_jwt_secret,omitempty" yaml:"state_jwt_secret" msg:"state_jwt_secret"`
	AllowedCallbackURLs            []string                     `json:"allowed_callback_urls,omitempty" yaml:"allowed_callback_urls" msg:"allowed_callback_urls"`
	ExternalAccessTokenFlowEnabled bool                         `json:"external_access_token_flow_enabled,omitempty" yaml:"external_access_token_flow_enabled" msg:"external_access_token_flow_enabled"`
	OnUserDuplicateAllowMerge      bool                         `json:"on_user_duplicate_allow_merge,omitempty" yaml:"on_user_duplicate_allow_merge" msg:"on_user_duplicate_allow_merge"`
	OnUserDuplicateAllowCreate     bool                         `json:"on_user_duplicate_allow_create,omitempty" yaml:"on_user_duplicate_allow_create" msg:"on_user_duplicate_allow_create"`
	Providers                      []OAuthProviderConfiguration `json:"providers,omitempty" yaml:"providers" msg:"providers"`
}

func (s *OAuthConfiguration) APIEndpoint() string {
	// URLPrefix can't be seen as skygear endpoint.
	// Consider URLPrefix = http://localhost:3001/auth
	// and skygear SDK use is as base endpint URL (in iframe_html and auth_handler_html).
	// And then, SDK may generate wrong action path base on this wrong endpoint (http://localhost:3001/auth).
	// So, this function will remote path part of URLPrefix
	u, err := url.Parse(s.URLPrefix)
	if err != nil {
		return s.URLPrefix
	}
	u.Path = ""
	return u.String()
}

type OAuthProviderType string

const (
	OAuthProviderTypeGoogle    OAuthProviderType = "google"
	OAuthProviderTypeFacebook  OAuthProviderType = "facebook"
	OAuthProviderTypeInstagram OAuthProviderType = "instagram"
	OAuthProviderTypeLinkedIn  OAuthProviderType = "linkedin"
	OAuthProviderTypeAzureADv2 OAuthProviderType = "azureadv2"
)

type OAuthProviderConfiguration struct {
	ID           string            `json:"id,omitempty" yaml:"id" msg:"id"`
	Type         OAuthProviderType `json:"type,omitempty" yaml:"type" msg:"type"`
	ClientID     string            `json:"client_id,omitempty" yaml:"client_id" msg:"client_id"`
	ClientSecret string            `json:"client_secret,omitempty" yaml:"client_secret" msg:"client_secret"`
	Scope        string            `json:"scope,omitempty" yaml:"scope" msg:"scope"`
	// Type specific fields
	Tenant string `json:"tenant,omitempty" yaml:"tenant" msg:"tenant"`
}

type UserVerificationCriteria string

const (
	// Some login ID need to verified belonging to the user is verified
	UserVerificationCriteriaAny UserVerificationCriteria = "any"
	// All login IDs need to verified belonging to the user is verified
	UserVerificationCriteriaAll UserVerificationCriteria = "all"
)

func (criteria UserVerificationCriteria) IsValid() bool {
	return criteria == UserVerificationCriteriaAny || criteria == UserVerificationCriteriaAll
}

type UserVerificationConfiguration struct {
	URLPrefix        string                                      `json:"url_prefix,omitempty" yaml:"url_prefix" msg:"url_prefix"`
	AutoSendOnSignup bool                                        `json:"auto_send_on_signup,omitempty" yaml:"auto_send_on_signup" msg:"auto_send_on_signup"`
	Criteria         UserVerificationCriteria                    `json:"criteria,omitempty" yaml:"criteria" msg:"criteria"`
	ErrorRedirect    string                                      `json:"error_redirect,omitempty" yaml:"error_redirect" msg:"error_redirect"`
	ErrorHTMLURL     string                                      `json:"error_html_url,omitempty" yaml:"error_html_url" msg:"error_html_url"`
	LoginIDKeys      map[string]UserVerificationKeyConfiguration `json:"login_id_keys,omitempty" yaml:"login_id_keys" msg:"login_id_keys"`
}

type UserVerificationCodeFormat string

const (
	UserVerificationCodeFormatNumeric UserVerificationCodeFormat = "numeric"
	UserVerificationCodeFormatComplex UserVerificationCodeFormat = "complex"
)

func (format UserVerificationCodeFormat) IsValid() bool {
	return format == UserVerificationCodeFormatNumeric || format == UserVerificationCodeFormatComplex
}

type UserVerificationProvider string

const (
	UserVerificationProviderSMTP   UserVerificationProvider = "smtp"
	UserVerificationProviderTwilio UserVerificationProvider = "twilio"
	UserVerificationProviderNexmo  UserVerificationProvider = "nexmo"
)

func (format UserVerificationProvider) IsValid() bool {
	switch format {
	case UserVerificationProviderSMTP:
		return true
	case UserVerificationProviderTwilio:
		return true
	case UserVerificationProviderNexmo:
		return true
	}
	return false
}

type UserVerificationKeyConfiguration struct {
	CodeFormat      UserVerificationCodeFormat            `json:"code_format,omitempty" yaml:"code_format" msg:"code_format"`
	Expiry          int64                                 `json:"expiry,omitempty" yaml:"expiry" msg:"expiry"`
	SuccessRedirect string                                `json:"success_redirect,omitempty" yaml:"success_redirect" msg:"success_redirect"`
	SuccessHTMLURL  string                                `json:"success_html_url,omitempty" yaml:"success_html_url" msg:"success_html_url"`
	ErrorRedirect   string                                `json:"error_redirect,omitempty" yaml:"error_redirect" msg:"error_redirect"`
	ErrorHTMLURL    string                                `json:"error_html_url,omitempty" yaml:"error_html_url" msg:"error_html_url"`
	Provider        UserVerificationProvider              `json:"provider,omitempty" yaml:"provider" msg:"provider"`
	ProviderConfig  UserVerificationProviderConfiguration `json:"provider_config,omitempty" yaml:"provider_config" msg:"provider_config"`
}

type UserVerificationProviderConfiguration struct {
	Subject     string `json:"subject,omitempty" yaml:"subject" msg:"subject"`
	Sender      string `json:"sender,omitempty" yaml:"sender" msg:"sender"`
	SenderName  string `json:"sender_name,omitempty" yaml:"sender_name" msg:"sender_name"`
	ReplyTo     string `json:"reply_to,omitempty" yaml:"reply_to" msg:"reply_to"`
	ReplyToName string `json:"reply_to_name,omitempty" yaml:"reply_to_name" msg:"reply_to_name"`
	TextURL     string `json:"text_url,omitempty" yaml:"text_url" msg:"text_url"`
	HTMLURL     string `json:"html_url,omitempty" yaml:"html_url" msg:"html_url"`
}

type HookUserConfiguration struct {
	Secret string `json:"secret,omitempty" yaml:"secret" msg:"secret"`
}

// AppConfiguration is configuration kept secret from the developer.
type AppConfiguration struct {
	DatabaseURL string               `json:"database_url,omitempty" yaml:"database_url" msg:"database_url"`
	SMTP        SMTPConfiguration    `json:"smtp,omitempty" yaml:"smtp" msg:"smtp"`
	Twilio      TwilioConfiguration  `json:"twilio,omitempty" yaml:"twilio" msg:"twilio"`
	Nexmo       NexmoConfiguration   `json:"nexmo,omitempty" yaml:"nexmo" msg:"nexmo"`
	Hook        HookAppConfiguration `json:"hook,omitempty" yaml:"hook" msg:"hook"`
}

type SMTPMode string

const (
	SMTPModeNormal SMTPMode = "normal"
	SMTPModeSSL    SMTPMode = "ssl"
)

func (mode SMTPMode) IsValid() bool {
	switch mode {
	case SMTPModeNormal:
		return true
	case SMTPModeSSL:
		return true
	}
	return false
}

type SMTPConfiguration struct {
	Host     string   `json:"host,omitempty" yaml:"host" msg:"host"`
	Port     int      `json:"port,omitempty" yaml:"port" msg:"port"`
	Mode     SMTPMode `json:"mode,omitempty" yaml:"mode" msg:"mode"`
	Login    string   `json:"login,omitempty" yaml:"login" msg:"login"`
	Password string   `json:"password,omitempty" yaml:"password" msg:"password"`
}

type TwilioConfiguration struct {
	AccountSID string `json:"account_sid,omitempty" yaml:"account_sid" msg:"account_sid"`
	AuthToken  string `json:"auth_token,omitempty" yaml:"auth_token" msg:"auth_token"`
	From       string `json:"from,omitempty" yaml:"from" msg:"from"`
}

type NexmoConfiguration struct {
	APIKey    string `json:"api_key,omitempty" yaml:"api_key" msg:"api_key"`
	APISecret string `json:"secret,omitempty" yaml:"secret" msg:"secret"`
	From      string `json:"from,omitempty" yaml:"from" msg:"from"`
}

type HookAppConfiguration struct {
	SyncHookTimeout      int `json:"sync_hook_timeout_second,omitempty" yaml:"sync_hook_timeout_second" msg:"sync_hook_timeout_second"`
	SyncHookTotalTimeout int `json:"sync_hook_total_timeout_second,omitempty" yaml:"sync_hook_total_timeout_second" msg:"sync_hook_total_timeout_second"`
}

var (
	_ sql.Scanner   = &TenantConfiguration{}
	_ driver.Valuer = &TenantConfiguration{}
)
