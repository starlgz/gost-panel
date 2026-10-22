// GOST 运行状态接口
import api from '@/api'

export const statusApi = {
  get() {
    return api.get('/status')
  },
}
