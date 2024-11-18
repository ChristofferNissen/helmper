package commands

// import (
// 	"fmt"
// 	"os"

// 	homedir "github.com/mitchellh/go-homedir"
// 	"github.com/spf13/cobra"
// 	"github.com/spf13/viper"
// )

// func init() {
// 	cobra.OnInitialize(initConfig)
// 	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cobra.yaml)")
// 	rootCmd.PersistentFlags().StringVarP(&projectBase, "projectbase", "b", "", "base project directory eg. github.com/spf13/")
// 	rootCmd.PersistentFlags().StringP("author", "a", "YOUR NAME", "Author name for copyright attribution")
// 	rootCmd.PersistentFlags().StringVarP(&userLicense, "license", "l", "", "Name of license for the project (can provide `licensetext` in config)")
// 	rootCmd.PersistentFlags().Bool("viper", true, "Use Viper for configuration")
// 	viper.BindPFlag("author", rootCmd.PersistentFlags().Lookup("author"))
// 	viper.BindPFlag("projectbase", rootCmd.PersistentFlags().Lookup("projectbase"))
// 	viper.BindPFlag("useViper", rootCmd.PersistentFlags().Lookup("viper"))
// 	viper.SetDefault("author", "NAME HERE <EMAIL ADDRESS>")
// 	viper.SetDefault("license", "apache")
// }

// func initConfig() {
// 	// Don't forget to read config either from cfgFile or from home directory!
// 	if cfgFile != "" {
// 		// Use config file from the flag.
// 		viper.SetConfigFile(cfgFile)
// 	} else {
// 		// Find home directory.
// 		home, err := homedir.Dir()
// 		if err != nil {
// 			fmt.Println(err)
// 			os.Exit(1)
// 		}

// 		// Search config in home directory with name ".cobra" (without extension).
// 		viper.AddConfigPath(home)
// 		viper.SetConfigName(".cobra")
// 	}

// 	if err := viper.ReadInConfig(); err != nil {
// 		fmt.Println("Can't read config:", err)
// 		os.Exit(1)
// 	}
// }
