package config

import (
	"fmt"

	"github.com/opsmx/ai-guardian-api/pkg/auth/session"
)

// Login Session Store: Cookie store is the default, we can use redis on postres (oes-db)
func AuthenticatorSessionStore() error {
	// Session timeout and backend store
	switch config.SessionStoreType {
	case "memory":
		session.InitMemSessionsStore(int(SessionTimeout))
	case "redis":
		err := session.InitRedisSessionsStore(int(SessionTimeout), config.Redis.Address, config.Redis.User, config.Redis.Password)
		if err != nil {
			msg := fmt.Sprintf("error initializing redis session store: %v", err)
			config.StartUpMessages = append(config.StartUpMessages, msg)
			config.HomePage = "/diagnostics"
			return err
		}
	case "postgres":
		pg := config.Pg
		err := session.InitPgSessionsStore(int(SessionTimeout), pg.User, pg.Password, pg.Address, pg.Database, pg.SSLMode)
		if err != nil {
			msg := fmt.Sprintf("error initializing postgres session store: %v", err)
			config.StartUpMessages = append(config.StartUpMessages, msg)
			config.HomePage = "/diagnostics"
			return err
		}
	default:
		msg := "unknown session storage type, using memory. Valid values are: memory, redis and postgres"
		config.StartUpMessages = append(config.StartUpMessages, msg)
		session.InitMemSessionsStore(int(SessionTimeout))
	}
	return nil
}
