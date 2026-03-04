package icinga2

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/legioner0/go-icinga2-api/iapi"
)

var (
	errInsecureSSL = errors.New("Requests are only allowed to use the HTTPS protocol so that traffic remains encrypted")
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ICINGA2_API_URL", nil),
				Description: "The address of the Icinga2 server.",
			},
			"api_user": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ICINGA2_API_USER", nil),
				Description: "The user to authenticate to the Icinga2 Server as.",
			},
			"api_password": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("ICINGA2_API_PASSWORD", nil),
				Description: "The password for authenticating to the Icinga2 server.",
			},
			"insecure_skip_tls_verify": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: EnvBoolDefaultFunc("ICINGA2_INSECURE_SKIP_TLS_VERIFY", false),
				Description: "Disable TLS verify when connecting to Icinga2 Server.",
			},
			"ca_cert_file": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ICINGA2_API_CA_CERT_FILE", ""),
				Description: "The CA certificate of Icinga2 Server.",
			},
			"retries": {
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ICINGA2_RETRIES", 0),
				Description: "How many times to retry on low level errors and `503 Icinga is reloading`. Defaults to `0`.\n",
			},
			"retry_delay": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ICINGA2_RETRY_DELAY", "0"),
				Description: "Delay between retry attempts. Valid values are durations expressed as `500ms`, etc. or a plain number which is treated as whole seconds.\n",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"icinga2_host":         resourceIcinga2Host(),
			"icinga2_checkcommand": resourceIcinga2Checkcommand(),
			"icinga2_service":      resourceIcinga2Service(),
			"icinga2_user":         resourceIcinga2User(),
			"icinga2_notification": resourceIcinga2Notification(),
		},
		ConfigureFunc: configureProvider,
	}
}

func configureProvider(d *schema.ResourceData) (interface{}, error) {
	delay := d.Get("retry_delay").(string)
	var duration time.Duration

	if delay != "" {
		var err error
		// Try parsing as a duration
		duration, err = time.ParseDuration(delay)
		if err != nil {
			// Failing that, convert to an integer and treat as seconds
			seconds, err := strconv.Atoi(delay)
			if err != nil {
				return nil, fmt.Errorf("invalid delay: %s", delay)
			}
			duration = time.Duration(seconds) * time.Second
		}
		if duration < 0 {
			return nil, fmt.Errorf("delay cannot be negative: %s", duration)
		}
	}

	caCertFile := d.Get("ca_cert_file").(string)
	if caCertFile != "" {
		_, err := os.Stat(caCertFile)
		if err != nil {
			return nil, err
		}
	}

	config, _ := iapi.New(
		d.Get("api_user").(string),
		d.Get("api_password").(string),
		d.Get("api_url").(string),
		d.Get("insecure_skip_tls_verify").(bool),
		caCertFile,
		int32(d.Get("retries").(int)),
		duration,
	)

	if err := validateURL(d.Get("api_url").(string)); err != nil {
		return nil, err
	}

	if err, _ := config.Connect(); err != nil {
		return nil, err
	}

	return config, nil
}

func validateURL(urlString string) error {
	tokens, err := url.Parse(urlString)
	if err != nil {
		return err
	}

	if tokens.Scheme != "https" {
		return errInsecureSSL
	}

	if !strings.HasSuffix(tokens.Path, "/v1") {
		return fmt.Errorf("error : Invalid API version %s specified. Only v1 is currently supported", tokens.Path)
	}

	return nil
}

// EnvBoolDefaultFunc is a helper function that returns
func EnvBoolDefaultFunc(k string, dv interface{}) schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		if v := os.Getenv(k); v == "true" {
			return true, nil
		}

		return false, nil
	}
}
