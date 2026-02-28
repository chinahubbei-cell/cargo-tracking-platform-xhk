# 全球港口与机场数据导入工具

## 目录结构

```
data_import/
├── import_airports.go      # 机场数据导入脚本
├── update_airports.sh      # 自动更新脚本
├── data_sources/
│   ├── airports.csv        # OurAirports 机场数据
│   └── backups/            # 备份目录
└── README.md               # 本文件
```

## 数据源

### 机场数据
- **来源**: [OurAirports](https://ourairports.com/data/)
- **许可**: Public Domain (免费)
- **数据量**: 84,000+ 机场

### 港口数据
- 内置中欧班列铁路港 25 个
- 全球主要海港 57 个

## 使用方法

### 手动导入机场数据

```bash
cd trackcard-server/scripts/data_import

# 1. 下载最新数据 (如需更新)
curl -L -o data_sources/airports.csv \
    "https://davidmegginson.github.io/ourairports-data/airports.csv"

# 2. 执行导入
go run import_airports.go
```

### 自动更新 (每周)

设置 crontab 定时任务：

```bash
# 编辑 crontab
crontab -e

# 添加以下行 (每周日凌晨3点执行)
0 3 * * 0 /path/to/trackcard-server/scripts/data_import/update_airports.sh
```

或手动执行：

```bash
./update_airports.sh
```

## 导入结果

导入后数据库统计：

| 类型 | 数量 | 覆盖国家 |
|------|------|----------|
| 机场 | 4,524 | 236 |
| 港口 | 82 | 47 |
| 铁路港 | 25 | 11 |

## 注意事项

1. 导入脚本使用 **upsert** 逻辑，不会删除现有数据
2. 只导入有 IATA 代码的大中型机场
3. 主要货运枢纽会自动标记 `is_cargo_hub = true`
4. 备份文件保留最近 7 个

## 日志

更新日志保存在 `update_airports.log`
