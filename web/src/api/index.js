// 统一导出所有 API 域模块
export { authApi } from '@/api/auth'
export { rulesApi } from '@/api/rules'
export { statusApi } from '@/api/status'
export { auditApi } from '@/api/audit'
export { default as api, ApiError, handleError } from '@/api'
