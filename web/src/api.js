// axios 实例：统一 baseURL、认证头、响应解包、错误处理
// 后端统一响应格式：{"code":0,"message":"ok","data":...}（失败 code!=0）
import axios from 'axios'

const api = axios.create({ baseURL: '/api' })

api.interceptors.request.use((cfg) => {
  const token = localStorage.getItem('token')
  if (token) cfg.headers.Authorization = `Bearer ${token}`
  return cfg
})

// 响应拦截：code==0 时直接返回 data（视图层拿到的就是业务数据），
// code!=0 时抛出带 message 的错误
api.interceptors.response.use(
  (res) => {
    const body = res.data
    if (body && typeof body === 'object' && 'code' in body) {
      if (body.code !== 0) {
        const apiErr = new ApiError(body.message || '请求失败', body.code)
        // 业务码 40100 = 未认证，自动跳登录
        if (body.code === 40100) {
          localStorage.removeItem('token')
          if (location.hash !== '#/login') {
            location.hash = '#/login'
          }
        }
        return Promise.reject(apiErr)
      }
      res.data = body.data
    }
    return res
  },
  (err) => Promise.reject(normalizeErr(err)),
)

// ApiError 携带业务码与用户可读消息。
export class ApiError extends Error {
  constructor(message, code) {
    super(message)
    this.name = 'ApiError'
    this.code = code
  }
}

function normalizeErr(err) {
  // HTTP 层错误（4xx/5xx）：后端 body 也是统一格式
  const body = err?.response?.data
  const apiErr = body && typeof body === 'object' && 'code' in body
    ? new ApiError(body.message || '请求失败', body.code)
    : new ApiError(err?.message || '请求失败', -1)

  // 401 = token 失效或未登录：清 token 并跳回登录页
  if (err?.response?.status === 401 || apiErr.code === 40100) {
    localStorage.removeItem('token')
    // 避免在登录页本身重复跳转
    if (location.hash !== '#/login') {
      location.hash = '#/login'
    }
  }
  return apiErr
}

// handleError 把任意错误转成用户可读文案（视图 flash 用）。
export function handleError(e) {
  return e instanceof Error ? e.message : '请求失败'
}

export default api
