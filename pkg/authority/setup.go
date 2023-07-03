package authority

import (
	"fmt"
	"github.com/apache/dubbo-admin/pkg/authority/config"
	"github.com/apache/dubbo-admin/pkg/authority/security"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (
	// For example, --webhook-port is bound to DUBBO_WEBHOOK_PORT.
	envNamePrefix = "DUBBO"

	// Replace hyphenated flag names with camelCase
	replaceWithCamelCase = false
)

func Run(options *config.Options) error {
	s := security.NewServer(options)

	s.Init()
	s.Start()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	signal.Notify(s.StopChan, syscall.SIGINT, syscall.SIGTERM)
	signal.Notify(s.CertStorage.GetStopChan(), syscall.SIGINT, syscall.SIGTERM)

	<-c

	return nil
}

func Initialize(cmd *cobra.Command) error {
	v := viper.New()

	// For example, --webhook-port is bound to DUBBO_WEBHOOK_PORT.
	v.SetEnvPrefix(envNamePrefix)

	// keys with underscores, e.g. DUBBO-WEBHOOK-PORT to DUBBO_WEBHOOK_PORT
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Bind to environment variables
	v.AutomaticEnv()

	// Bind the current command's flags to viper
	bindFlags(cmd, v)

	return nil
}

func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		configName := f.Name

		//  Replace hyphens with a camelCased string.
		if replaceWithCamelCase {
			configName = strings.ReplaceAll(f.Name, "-", "")
		}

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(configName) {
			val := v.Get(configName)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}
