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
import { TASK_ACTIONS, TASK_STATUS } from '../constants'
import type { TaskLog } from '../types'

export function buildPublicVideoContentURL(
  taskId: string,
  origin?: string
): string {
  const path = `/v1/videos/${encodeURIComponent(taskId)}/content`
  return origin ? new URL(path, origin).href : path
}

export function buildPublicVideoDownloadURL(
  taskId: string,
  origin?: string
): string {
  const contentURL = buildPublicVideoContentURL(taskId, origin)
  return `${contentURL}?download_name=${encodeURIComponent(taskId)}`
}

export function isVideoTask(log: TaskLog): boolean {
  return (
    log.action === TASK_ACTIONS.GENERATE ||
    log.action === TASK_ACTIONS.TEXT_GENERATE ||
    log.action === TASK_ACTIONS.FIRST_TAIL_GENERATE ||
    log.action === TASK_ACTIONS.REFERENCE_GENERATE ||
    log.action === TASK_ACTIONS.REMIX_GENERATE
  )
}

export function canUploadVideoTaskToS3(log: TaskLog): boolean {
  return (
    log.status === TASK_STATUS.SUCCESS &&
    isVideoTask(log) &&
    !['available', 'uploading'].includes(log.video_storage_status || '')
  )
}
