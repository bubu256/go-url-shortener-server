package config

// здесь пока очень пусто, но наверное в будущем эта структура пригодится

func New() Configuration {
	return Configuration{Server: CfgServer{Port: "8080"}}
}

type Configuration struct {
	DB     CfgDataBase
	Server CfgServer
	// ... тут будут конфиги и для других модулей наверное
}

type CfgDataBase struct {
	InitialData map[string]string
	// ... тут будут настройки для Базы данных
}

type CfgServer struct {
	Port string
	// ... тут будут остальные настройки для Сервера
}
