// 认证相关接口
import api from '../api'

export const authApi = {
  login(username, password) {
    return api.post('/login', { username, password })
  },

  changePassword(oldPassword, newPassword) {
    return api.post('/change-password', {
      old_password: oldPassword,
      new_password: newPassword,
    })
  },
}
