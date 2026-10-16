// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zsvirt

import (
	"context"
	"crypto/sha512"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

const defaultClientLoginTimeout = 30 * time.Second

type AccessConfig struct {
	Host            string `mapstructure:"management_ip"`
	LegacyHost      string `mapstructure:"zstack_host"`
	Port            int    `mapstructure:"port"`
	AccountName     string `mapstructure:"account_name"`
	AccountPassword string `mapstructure:"account_password"`
	AccessKeyId     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	ctx             interpolate.Context

	portEnvErr error
}

// firstEnv returns the first non-empty value and its variable name.
func firstEnv(names ...string) (string, string) {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value, name
		}
	}
	return "", ""
}

// resolveAlias preserves legacy templates, rejecting ambiguous configuration.
func resolveAlias(value *string, legacy, name, legacyName string) error {
	if *value != "" && legacy != "" && *value != legacy {
		return fmt.Errorf("%s and deprecated %s must agree when both are configured", name, legacyName)
	}
	if *value == "" {
		*value = legacy
	}
	return nil
}

func (c *AccessConfig) applyEnvDefaults() {
	c.portEnvErr = nil
	if c.Host == "" {
		c.Host = c.LegacyHost
	}
	if c.Host == "" {
		c.Host, _ = firstEnv("ZSVIRT_MANAGEMENT_IP", "ZSPHERE_HOST", "ZSTACK_HOST")
	}
	if c.Port == 0 {
		if raw, name := firstEnv("ZSVIRT_PORT", "ZSPHERE_PORT", "ZSTACK_PORT"); raw != "" {
			p, err := strconv.Atoi(raw)
			if err != nil {
				c.portEnvErr = fmt.Errorf("%s is not a valid integer: %q", name, raw)
			} else {
				c.Port = p
			}
		}
	}
	// Explicit credentials form one group. Never fill a partial group from
	// another source or combine account and access-key modes across namespaces.
	if c.AccountName != "" || c.AccountPassword != "" || c.AccessKeyId != "" || c.AccessKeySecret != "" {
		return
	}
	for _, prefix := range []string{"ZSVIRT_", "ZSPHERE_", "ZSTACK_"} {
		account, password := os.Getenv(prefix+"ACCOUNT_NAME"), os.Getenv(prefix+"ACCOUNT_PASSWORD")
		key, secret := os.Getenv(prefix+"ACCESS_KEY_ID"), os.Getenv(prefix+"ACCESS_KEY_SECRET")
		if account != "" || password != "" || key != "" || secret != "" {
			c.AccountName, c.AccountPassword = account, password
			c.AccessKeyId, c.AccessKeySecret = key, secret
			return
		}
	}
}

func (c *AccessConfig) validateCredentials() []error {
	var errs []error
	if err := resolveAlias(&c.Host, c.LegacyHost, "management_ip", "zstack_host"); err != nil {
		errs = append(errs, err)
	}
	if c.portEnvErr != nil {
		errs = append(errs, c.portEnvErr)
	}
	if c.Host == "" {
		errs = append(errs, fmt.Errorf("host is required"))
	}

	if (c.AccountName == "" || c.AccountPassword == "") && (c.AccessKeyId == "" || c.AccessKeySecret == "") {
		errs = append(errs, fmt.Errorf("either account_name + account_password or access_key_id + access_key_secret is required"))
	}

	if c.Port < 0 || c.Port > 65535 {
		errs = append(errs, fmt.Errorf("port must be between 1 and 65535"))
	}
	if (c.AccountName != "" || c.AccountPassword != "") && (c.AccessKeyId != "" || c.AccessKeySecret != "") {
		errs = append(errs, fmt.Errorf("account and access key authentication are mutually exclusive"))
	}
	return errs
}

func (c *AccessConfig) Prepare(raws ...interface{}) []error {
	c.ctx = interpolate.Context{}

	var errs []error
	err := config.Decode(c, &config.DecodeOpts{
		Interpolate:        true,
		InterpolateContext: &c.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{
				"run_command",
			},
		},
	}, raws...)
	if err != nil {
		errs = append(errs, err)
	}

	c.applyEnvDefaults()
	errs = append(errs, c.validateCredentials()...)

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (c *AccessConfig) CreateClient(debug bool) (*apiClient, error) {
	return c.createClient(context.Background())
}
func (c *AccessConfig) createClient(parent context.Context) (*apiClient, error) {
	if errs := c.validateCredentials(); len(errs) > 0 {
		return nil, errs[0]
	}
	cli, err := newAPIClient(c)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(parent, defaultClientLoginTimeout)
	defer cancel()
	if c.AccountName != "" && c.AccountPassword != "" {
		if _, err := cli.Login(ctx); err != nil {
			return nil, redactError(err, c.secretValues())
		}
		return cli, nil
	}
	d := &ZSvirtDriver{client: cli, ctx: ctx, secrets: c.secretValues()}
	if err := d.ValidateCredentials(); err != nil {
		return nil, err
	}
	return cli, nil
}

func (c *AccessConfig) secretValues() []string {
	values := []string{c.AccountPassword, c.AccessKeyId, c.AccessKeySecret}
	if c.AccountPassword != "" {
		values = append(values, fmt.Sprintf("%x", sha512.Sum512([]byte(c.AccountPassword))))
	}
	return values
}
func (c *AccessConfig) Driver(debug bool) (*ZSvirtDriver, error) {
	return c.driver(context.Background(), defaultAPITimeout)
}
func (c *AccessConfig) driver(ctx context.Context, timeout time.Duration) (*ZSvirtDriver, error) {
	cli, err := c.createClient(ctx)
	if err != nil {
		return nil, err
	}
	return &ZSvirtDriver{client: cli, ctx: ctx, timeout: timeout, secrets: append(c.secretValues(), cli.session)}, nil
}
