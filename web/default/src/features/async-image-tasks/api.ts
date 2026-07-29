/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { api } from '@/lib/api'

import type {
  ApiResponse,
  AsyncImageTaskDetail,
  AsyncImageTaskList,
  RetryResult,
} from './types'

export type AsyncImageTaskFilters = {
  page: number
  page_size: number
  task_id?: string
  model?: string
  status?: string
  output_availability?: string
  billing_status?: string
}

export async function getAsyncImageTasks(
  filters: AsyncImageTaskFilters,
  root: boolean
) {
  const endpoint = root ? '/api/async-image-task' : '/api/async-image-task/self'
  const response = await api.get<ApiResponse<AsyncImageTaskList>>(endpoint, {
    params: filters,
  })
  return response.data.data
}

export async function getAsyncImageTaskDetail(taskId: string, root: boolean) {
  const endpoint = root
    ? `/api/async-image-task/${encodeURIComponent(taskId)}`
    : `/api/async-image-task/self/${encodeURIComponent(taskId)}`
  const response = await api.get<ApiResponse<AsyncImageTaskDetail>>(endpoint)
  return response.data.data
}

export async function retryAsyncImageTasks(taskIds: string[]) {
  const response = await api.post<ApiResponse<RetryResult>>(
    '/api/async-image-task/retry',
    { task_ids: taskIds }
  )
  return response.data.data
}

export async function retryAllFailedAsyncImageTasks() {
  const response = await api.post<ApiResponse<{ operation_id: string }>>(
    '/api/async-image-task/retry-failed'
  )
  return response.data.data
}
