package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestUnmarshalMysqlYaml(t *testing.T) {
	yml := `
default: 'core'
instance:
  core:
    host: '127.0.0.1'
    port: '8888'
    user: 'root'
    password: '1234567'
    database: 'ca_user'
    debug: true
    open: 100
    idle: 20
    max_life_time: 3600
    max_idle_time: 300
    driver: 'mysql'
    ssl_config:
      enable: true
      path: '/ssl/core.pem'
    sharding:
      - table_name: 'b_user'
        sharding_key: 'user_id'
        sharding_num: 100
        p_k_generator: 0
    # 雪花 id配置
    snowflake_gen:
      # 是否使用默认的雪花 id数据中心
      enable_default: true
      snowflakes:
        - data_center: 'center_user'
          worker_id_bit: 10
          table_name: # 在这个数据中心下的表
            - 'b_user' 
            - 'b_user_logs'
    # 从库配置
    replicas:
      - host: '127.0.0.12'
        port: '8888'
        user: 'root'
        password: '1234567'
        database: 'ca_user'
        debug: true
        open: 100
        idle: 20
        max_life_time: 3600
        max_idle_time: 300
        driver: 'mysql'
        ssl_config:
          enable: true
          path: '/ssl/core.pem'
        sharding:
          - table_name: 'b_user'
            sharding_key: 'user_id'
            sharding_num: 100
            p_k_generator: 0
  game:
    host: '127.0.0.13'
    port: '8888'
    user: 'root'
    password: '1234567'
    database: 'ca_user'
    debug: true
    open: 100
    idle: 20
    max_life_time: 3600
    max_idle_time: 300
    driver: 'mysql'
    ssl_config:
      enable: true
      path: '/ssl/core.pem'
    # 雪花 id配置
    snowflake_gen:
      # 是否使用默认的雪花 id数据中心
      enable_default: true
      snowflakes:
        - data_center: 'center_game_color'
          worker_id_bit: 10
          table_name: # 在这个数据中心下的表
            - 'b_color_order' 
            - 'b_color_bet'
        # 雪花 id配置
    snowflake_gen:
      # 是否使用默认的雪花 id数据中心
      enable_default: true
      # 开启了 默认数据中心 但是没配置 snowflakes，那么所有的表都会走默认的数据中心获取雪花 id
    # 从库配置
    replicas:
      - host: '127.0.0.14'
        port: '8888'
        user: 'root'
        password: '1234567'
        database: 'ca_user'
        debug: true
        open: 100
        idle: 20
        max_life_time: 3600
        max_idle_time: 300
        driver: 'mysql'
        ssl_config:
          enable: true
          path: '/ssl/core.pem'
        sharding:
          - table_name: 'b_user'
            sharding_key: 'user_id'
            sharding_num: 100
            p_k_generator: 0

  


`

	obj := new(DbConfig)
	err := yaml.Unmarshal([]byte(yml), obj)
	if err != nil {
		t.Error(err)
	}
	t.Logf("%+v\n", obj)
}
