package sharding

import (
	fmt "fmt"
	easy_facade "github.com/coderyw/easymap/gen/facade"
	strconv "strconv"
	unsafe "unsafe"
)

func str2Bytes_sharding_easymap(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}
func (v *shardingDst) UnMarshalMap(m map[string]string) error {
	var (
		ok  bool
		val string
	)
	if val, ok = m["table"]; ok {
		{
			pv := val
			v.table = string(pv)
		}
	}

	return nil
}
func (v *shardingDst) UnMarshalMapInterface(m map[string]interface{}) error {
	var (
		ok  bool
		val interface{}
	)
	if val, ok = m["table"]; ok {
		switch val.(type) {
		case string:
			v.table = string(val.(string))
		}
	}

	return nil
}

func (v *shardingDst) MarshalMap() (map[string]interface{}, error) {
	m := make(map[string]interface{})
	m["table"] = v.table
	return m, nil
}

func (v *shardingDst) MarshalMapString() (map[string]string, error) {
	m := make(map[string]string)
	m["table"] = fmt.Sprint(v.table)
	return m, nil
}

type shardingDstField string

func (v shardingDstField) MarshalBinary() (data []byte, err error) {
	return str2Bytes_sharding_easymap(string(v)), nil
}

const (
	shardingDst_table shardingDstField = "table"
	shardingDst_dst   shardingDstField = "dst"
)

func (v *ShardingMigrator) UnMarshalMap(m map[string]string) error {
	return nil
}
func (v *ShardingMigrator) UnMarshalMapInterface(m map[string]interface{}) error {
	var (
		ok  bool
		val interface{}
	)
	if val, ok = m["sharding"]; ok {
		var i interface{} = v.sharding
		if m1, ok := val.(map[string]interface{}); ok {
			if b, ok := i.(easy_facade.EasyMapInter); ok {
				if err := b.UnMarshalMapInterface(m1); err != nil {
					return err
				}
			}
		} else if m2, ok := val.(map[string]string); ok {
			if b, ok := i.(easy_facade.EasyMapString); ok {
				if err := b.UnMarshalMap(m2); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (v *ShardingMigrator) MarshalMap() (map[string]interface{}, error) {
	m := make(map[string]interface{})
	return m, nil
}

func (v *ShardingMigrator) MarshalMapString() (map[string]string, error) {
	m := make(map[string]string)
	return m, nil
}

type ShardingMigratorField string

func (v ShardingMigratorField) MarshalBinary() (data []byte, err error) {
	return str2Bytes_sharding_easymap(string(v)), nil
}

const (
	ShardingMigrator_Migrator  ShardingMigratorField = "Migrator"
	ShardingMigrator_sharding  ShardingMigratorField = "sharding"
	ShardingMigrator_dialector ShardingMigratorField = "dialector"
)

func (v *ShardingDialector) UnMarshalMap(m map[string]string) error {
	return nil
}
func (v *ShardingDialector) UnMarshalMapInterface(m map[string]interface{}) error {
	var (
		ok  bool
		val interface{}
	)
	if val, ok = m["sharding"]; ok {
		var i interface{} = v.sharding
		if m1, ok := val.(map[string]interface{}); ok {
			if b, ok := i.(easy_facade.EasyMapInter); ok {
				if err := b.UnMarshalMapInterface(m1); err != nil {
					return err
				}
			}
		} else if m2, ok := val.(map[string]string); ok {
			if b, ok := i.(easy_facade.EasyMapString); ok {
				if err := b.UnMarshalMap(m2); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (v *ShardingDialector) MarshalMap() (map[string]interface{}, error) {
	m := make(map[string]interface{})
	return m, nil
}

func (v *ShardingDialector) MarshalMapString() (map[string]string, error) {
	m := make(map[string]string)
	return m, nil
}

type ShardingDialectorField string

func (v ShardingDialectorField) MarshalBinary() (data []byte, err error) {
	return str2Bytes_sharding_easymap(string(v)), nil
}

const (
	ShardingDialector_Dialector ShardingDialectorField = "Dialector"
	ShardingDialector_sharding  ShardingDialectorField = "sharding"
)

func (v *Sharding) UnMarshalMap(m map[string]string) error {
	return nil
}
func (v *Sharding) UnMarshalMapInterface(m map[string]interface{}) error {
	var (
		ok  bool
		val interface{}
	)
	if val, ok = m["DB"]; ok {
		var i interface{} = v.DB
		if m1, ok := val.(map[string]interface{}); ok {
			if b, ok := i.(easy_facade.EasyMapInter); ok {
				if err := b.UnMarshalMapInterface(m1); err != nil {
					return err
				}
			}
		} else if m2, ok := val.(map[string]string); ok {
			if b, ok := i.(easy_facade.EasyMapString); ok {
				if err := b.UnMarshalMap(m2); err != nil {
					return err
				}
			}
		}
	}
	if val, ok = m["ConnPool"]; ok {
		var i interface{} = v.ConnPool
		if m1, ok := val.(map[string]interface{}); ok {
			if b, ok := i.(easy_facade.EasyMapInter); ok {
				if err := b.UnMarshalMapInterface(m1); err != nil {
					return err
				}
			}
		} else if m2, ok := val.(map[string]string); ok {
			if b, ok := i.(easy_facade.EasyMapString); ok {
				if err := b.UnMarshalMap(m2); err != nil {
					return err
				}
			}
		}
	}
	if val, ok = m["querys"]; ok {
		var i interface{} = v.querys
		if m1, ok := val.(map[string]interface{}); ok {
			if b, ok := i.(easy_facade.EasyMapInter); ok {
				if err := b.UnMarshalMapInterface(m1); err != nil {
					return err
				}
			}
		} else if m2, ok := val.(map[string]string); ok {
			if b, ok := i.(easy_facade.EasyMapString); ok {
				if err := b.UnMarshalMap(m2); err != nil {
					return err
				}
			}
		}
	}
	if val, ok = m["_config"]; ok {
		var i interface{} = v._config
		if m1, ok := val.(map[string]interface{}); ok {
			if b, ok := i.(easy_facade.EasyMapInter); ok {
				if err := b.UnMarshalMapInterface(m1); err != nil {
					return err
				}
			}
		} else if m2, ok := val.(map[string]string); ok {
			if b, ok := i.(easy_facade.EasyMapString); ok {
				if err := b.UnMarshalMap(m2); err != nil {
					return err
				}
			}
		}
	}
	if val, ok = m["mutex"]; ok {
		var i interface{} = v.mutex
		if m1, ok := val.(map[string]interface{}); ok {
			if b, ok := i.(easy_facade.EasyMapInter); ok {
				if err := b.UnMarshalMapInterface(m1); err != nil {
					return err
				}
			}
		} else if m2, ok := val.(map[string]string); ok {
			if b, ok := i.(easy_facade.EasyMapString); ok {
				if err := b.UnMarshalMap(m2); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (v *Sharding) MarshalMap() (map[string]interface{}, error) {
	m := make(map[string]interface{})
	return m, nil
}

func (v *Sharding) MarshalMapString() (map[string]string, error) {
	m := make(map[string]string)
	return m, nil
}

type ShardingField string

func (v ShardingField) MarshalBinary() (data []byte, err error) {
	return str2Bytes_sharding_easymap(string(v)), nil
}

const (
	Sharding_DB             ShardingField = "DB"
	Sharding_ConnPool       ShardingField = "ConnPool"
	Sharding_configs        ShardingField = "configs"
	Sharding_querys         ShardingField = "querys"
	Sharding_snowflakeNodes ShardingField = "snowflakeNodes"
	Sharding__config        ShardingField = "_config"
	Sharding__tables        ShardingField = "_tables"
	Sharding_mutex          ShardingField = "mutex"
)

func (v *ConnPool) UnMarshalMap(m map[string]string) error {
	return nil
}
func (v *ConnPool) UnMarshalMapInterface(m map[string]interface{}) error {
	var (
		ok  bool
		val interface{}
	)
	if val, ok = m["sharding"]; ok {
		var i interface{} = v.sharding
		if m1, ok := val.(map[string]interface{}); ok {
			if b, ok := i.(easy_facade.EasyMapInter); ok {
				if err := b.UnMarshalMapInterface(m1); err != nil {
					return err
				}
			}
		} else if m2, ok := val.(map[string]string); ok {
			if b, ok := i.(easy_facade.EasyMapString); ok {
				if err := b.UnMarshalMap(m2); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (v *ConnPool) MarshalMap() (map[string]interface{}, error) {
	m := make(map[string]interface{})
	return m, nil
}

func (v *ConnPool) MarshalMapString() (map[string]string, error) {
	m := make(map[string]string)
	return m, nil
}

type ConnPoolField string

func (v ConnPoolField) MarshalBinary() (data []byte, err error) {
	return str2Bytes_sharding_easymap(string(v)), nil
}

const (
	ConnPool_sharding ConnPoolField = "sharding"
	ConnPool_ConnPool ConnPoolField = "ConnPool"
)

func (v *Config) UnMarshalMap(m map[string]string) error {
	var (
		ok  bool
		val string
	)
	if val, ok = m["DoubleWrite"]; ok {
		if pv, err := strconv.ParseBool(val); err != nil {
			return err
		} else {
			v.DoubleWrite = bool(pv)
		}
	}
	if val, ok = m["ShardingKey"]; ok {
		{
			pv := val
			v.ShardingKey = string(pv)
		}
	}
	if val, ok = m["NumberOfShards"]; ok {
		if pv, err := strconv.ParseUint(val, 10, 64); err != nil {
			return err
		} else {
			v.NumberOfShards = uint(pv)
		}
	}
	if val, ok = m["tableFormat"]; ok {
		{
			pv := val
			v.tableFormat = string(pv)
		}
	}

	return nil
}
func (v *Config) UnMarshalMapInterface(m map[string]interface{}) error {
	var (
		ok  bool
		val interface{}
	)
	if val, ok = m["DoubleWrite"]; ok {
		switch val.(type) {
		case bool:
			v.DoubleWrite = bool(val.(bool))
		}
	}
	if val, ok = m["ShardingKey"]; ok {
		switch val.(type) {
		case string:
			v.ShardingKey = string(val.(string))
		}
	}
	if val, ok = m["NumberOfShards"]; ok {
		switch val.(type) {
		case uint32:
			v.NumberOfShards = uint(val.(uint32))
		case uint64:
			v.NumberOfShards = uint(val.(uint64))
		case int16:
			v.NumberOfShards = uint(val.(int16))
		case int32:
			v.NumberOfShards = uint(val.(int32))
		case uint:
			v.NumberOfShards = uint(val.(uint))
		case uint8:
			v.NumberOfShards = uint(val.(uint8))
		case uint16:
			v.NumberOfShards = uint(val.(uint16))
		case int:
			v.NumberOfShards = uint(val.(int))
		case int8:
			v.NumberOfShards = uint(val.(int8))
		case int64:
			v.NumberOfShards = uint(val.(int64))
		}
	}
	if val, ok = m["tableFormat"]; ok {
		switch val.(type) {
		case string:
			v.tableFormat = string(val.(string))
		}
	}

	return nil
}

func (v *Config) MarshalMap() (map[string]interface{}, error) {
	m := make(map[string]interface{})
	m["DoubleWrite"] = v.DoubleWrite
	m["ShardingKey"] = v.ShardingKey
	m["NumberOfShards"] = v.NumberOfShards
	m["tableFormat"] = v.tableFormat
	m["PrimaryKeyGenerator"] = v.PrimaryKeyGenerator
	return m, nil
}

func (v *Config) MarshalMapString() (map[string]string, error) {
	m := make(map[string]string)
	m["DoubleWrite"] = fmt.Sprint(v.DoubleWrite)
	m["ShardingKey"] = fmt.Sprint(v.ShardingKey)
	m["NumberOfShards"] = fmt.Sprint(v.NumberOfShards)
	m["tableFormat"] = fmt.Sprint(v.tableFormat)
	m["PrimaryKeyGenerator"] = fmt.Sprint(v.PrimaryKeyGenerator)
	return m, nil
}

type ConfigField string

func (v ConfigField) MarshalBinary() (data []byte, err error) {
	return str2Bytes_sharding_easymap(string(v)), nil
}

const (
	Config_DoubleWrite                   ConfigField = "DoubleWrite"
	Config_ShardingKey                   ConfigField = "ShardingKey"
	Config_NumberOfShards                ConfigField = "NumberOfShards"
	Config_tableFormat                   ConfigField = "tableFormat"
	Config_ShardingAlgorithm             ConfigField = "ShardingAlgorithm"
	Config_ShardingSuffixs               ConfigField = "ShardingSuffixs"
	Config_ShardingAlgorithmByPrimaryKey ConfigField = "ShardingAlgorithmByPrimaryKey"
	Config_PrimaryKeyGenerator           ConfigField = "PrimaryKeyGenerator"
	Config_PrimaryKeyGeneratorFn         ConfigField = "PrimaryKeyGeneratorFn"
)
