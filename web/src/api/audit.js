// 审计日志相关接口
import api from '@/api'

export const auditApi = {
  list(params) {
    return api.get('/audit', { params })
  },

  clear() {
    return api.delete('/audit')
  },
}
