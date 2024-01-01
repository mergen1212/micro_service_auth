package config

import (
	"flag"
	
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {  
    // текущее окружение: local, dev, prod и т.п.
    Env            string     `yaml:"env" env-default:"local"`  
    // мы будем использовать SQLite, поэтому нужно указать путь до файла, где хранится наша БД
    StoragePath    string     `yaml:"storage_path" env-required:"true"`  
    //порт gRPC-сервиса и таймаут обработки запросов
    GRPC           GRPCConfig `yaml:"grpc"` 
    // путь до директории с миграциями БД. Он будет использоваться утилитой migrator 
    MigrationsPath string  
    // время жизни выдаваемых токенов авторизации.
    TokenTTL       time.Duration `yaml:"token_ttl" env-default:"1h"`  
}  

type GRPCConfig struct {  
    Port    int           `yaml:"port"`  
    Timeout time.Duration `yaml:"timeout"`  
}

func MustLoad() *Config {  
    configPath := fetchConfigPath()  
    if configPath == "" {  
        panic("config path is empty") 
    }  

    // check if file exists
    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        panic("config file does not exist: " + configPath)
    }

    var cfg Config

    if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
        panic("config path is empty: " + err.Error())
    }

    return &cfg
}

// fetchConfigPath fetches config path from command line flag or environment variable.
// Priority: flag > env > default.
// Default value is empty string.
func fetchConfigPath() string {
    var res string

    flag.StringVar(&res, "config", "", "path to config file")
    flag.Parse()

    if res == "" {
        res = os.Getenv("CONFIG_PATH")
    }
    
    return res
}

func MustLoadPath(configPath string) *Config {
	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("cannot read config: " + err.Error())
	}

	return &cfg
}