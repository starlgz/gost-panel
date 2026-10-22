// GOST 引擎相关接口
import api from '@/api'

export const gostApi = {
  // 从 gost 拉取当前运行的服务列表（和数据库规则做比对）
  services() {
    return api.get('/gost/services')
  },

  // 把 gost 运行的服务导入为面板规则
  importServices() {
    return api.post('/gost/import')
  },
}
