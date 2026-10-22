// 端口转发规则相关接口
import api from '@/api'

export const rulesApi = {
  stats() {
    return api.get('/rules/stats')
  },

  list(params) {
    return api.get('/rules', { params })
  },

  create(body) {
    return api.post('/rules', body)
  },

  update(id, body) {
    return api.put(`/rules/${id}`, body)
  },

  remove(id) {
    return api.delete(`/rules/${id}`)
  },

  toggleEnabled(id, enabled) {
    return api.put(`/rules/${id}`, { enabled })
  },

  apply() {
    return api.post('/rules/apply')
  },
}
