# 农机调度管理系统

**项目类型标签：全栈Web应用**

农机调度管理系统 是 农业合作社与种植大户 的农机作业调度平台。

## 快速启动

开发模式：

```bash
npm install
npm run dev
```

访问地址：`http://localhost:18621`

生产构建：

```bash
npm run build
npm run preview
```

## 主要功能

- 农机档案管理：编号、二维码、照片和状态筛选
- 作业任务调度：推荐空闲农机与驾驶员
- 实时地图轨迹：位置、轨迹回放和地块边界
- 作业统计报表：日报、月报和 Excel 导出
- 维修保养提醒：周期预警与记录追溯
- 驾驶员管理：证照、排班和评价
- 调度看板：待办、空闲、趋势和提醒

## 本地开发方式

在项目根目录执行：

```bash
npm install
npm run dev
```

## 技术栈

| 分类 | 技术 |
| --- | --- |
| 前端 | Vue 3 + TypeScript + Vite + Element Plus |
| 后端 | Spring Boot 3 + Java 17 |
| 数据库 | MySQL 8.0 |
| 缓存 | Redis |
| 认证 | JWT + Spring Security |

## 项目目录结构

```text
. 
├── src
│   ├── components
│   ├── constants
│   ├── data
│   ├── errors
│   ├── features
│   ├── logger
│   ├── services
│   ├── types
│   └── utils
├── Dockerfile
├── nginx.conf
├── package.json
└── README.md
```

## 环境变量说明

纯前端项目默认不需要后端环境变量。地图或第三方 API Key 可放入本地 .env 文件。

## 使用说明

应用数据存储在浏览器本地。清空浏览器站点数据会重置演示数据。

## License

MIT
