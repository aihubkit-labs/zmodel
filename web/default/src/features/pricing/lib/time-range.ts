/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

export type TimeRange = {
  start: string
  end: string
}

export type TimeRangeCondition = {
  variable: 'time_range'
  timezone: string
  ranges: TimeRange[]
}

export function timeToMinuteOfDay(value: string): number | null {
  const match = value.trim().match(/^(\d{2}):(\d{2})$/)
  if (!match) return null
  const hour = Number(match[1])
  const minute = Number(match[2])
  if (hour > 23 || minute > 59) return null
  return hour * 60 + minute
}

export function minuteOfDayToTime(value: string | number): string | null {
  const minutes = Number(value)
  if (!Number.isInteger(minutes) || minutes < 0 || minutes > 1439) {
    return null
  }
  const hour = Math.floor(minutes / 60)
  const minute = minutes % 60
  return `${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}`
}

export function buildTimeRangeConditionExpr(
  condition: TimeRangeCondition
): string | null {
  const timezone = condition.timezone.trim()
  if (!timezone || condition.ranges.length === 0) return null

  const timezoneLiteral = JSON.stringify(timezone)
  const minuteOfDay = `hour(${timezoneLiteral}) * 60 + minute(${timezoneLiteral})`
  const segments = condition.ranges.flatMap((range) => {
    const start = timeToMinuteOfDay(range.start)
    const end = timeToMinuteOfDay(range.end)
    if (start == null || end == null) return []
    const joiner = start <= end ? '&&' : '||'
    return [`(${minuteOfDay} >= ${start} ${joiner} ${minuteOfDay} <= ${end})`]
  })

  if (segments.length !== condition.ranges.length) return null
  return segments.length === 1 ? segments[0] : `(${segments.join(' || ')})`
}

function hasFullOuterParens(expr: string): boolean {
  if (!expr.startsWith('(') || !expr.endsWith(')')) return false
  let depth = 0
  for (let index = 0; index < expr.length; index += 1) {
    if (expr[index] === '(') depth += 1
    if (expr[index] === ')') depth -= 1
    if (depth === 0 && index < expr.length - 1) return false
  }
  return depth === 0
}

function unwrapOuterParens(expr: string): string {
  let current = expr.trim()
  while (hasFullOuterParens(current)) {
    current = current.slice(1, -1).trim()
  }
  return current
}

function splitTopLevel(expr: string, operator: '&&' | '||'): string[] {
  const parts: string[] = []
  let start = 0
  let depth = 0
  for (let index = 0; index < expr.length; index += 1) {
    const char = expr[index]
    if (char === '(') depth += 1
    if (char === ')') depth -= 1
    if (depth === 0 && expr.slice(index, index + 2) === operator) {
      parts.push(expr.slice(start, index).trim())
      start = index + 2
      index += 1
    }
  }
  parts.push(expr.slice(start).trim())
  return parts.filter(Boolean)
}

export function splitTopLevelAnd(expr: string): string[] {
  return splitTopLevel(expr, '&&')
}

function parseComparison(value: string) {
  const match = unwrapOuterParens(value).match(
    /^hour\("([^"]+)"\) \* 60 \+ minute\("([^"]+)"\) (>=|<=) (\d+)$/
  )
  if (!match || match[1] !== match[2]) return null
  return { timezone: match[1], operator: match[3], value: match[4] }
}

export function parseTimeRangeCondition(
  clause: string
): TimeRangeCondition | null {
  const segments = splitTopLevel(unwrapOuterParens(clause), '||')
  const ranges: TimeRange[] = []
  let timezone = ''

  // A cross-midnight range splits at the top-level OR, so parse it as one
  // range before handling the normal AND form and multiple windows.
  if (segments.length === 2) {
    const start = parseComparison(segments[0])
    const end = parseComparison(segments[1])
    if (
      start?.operator === '>=' &&
      end?.operator === '<=' &&
      start.timezone === end.timezone
    ) {
      const startTime = minuteOfDayToTime(start.value)
      const endTime = minuteOfDayToTime(end.value)
      if (startTime && endTime && startTime > endTime) {
        return {
          variable: 'time_range',
          timezone: start.timezone,
          ranges: [{ start: startTime, end: endTime }],
        }
      }
    }
  }

  for (const segment of segments) {
    const match = unwrapOuterParens(segment).match(
      /^hour\("([^"]+)"\) \* 60 \+ minute\("([^"]+)"\) >= (\d+) (&&|\|\|) hour\("([^"]+)"\) \* 60 \+ minute\("([^"]+)"\) <= (\d+)$/
    )
    if (
      !match ||
      match[1] !== match[2] ||
      match[1] !== match[5] ||
      match[1] !== match[6]
    ) {
      return null
    }
    if (timezone && timezone !== match[1]) return null

    const start = minuteOfDayToTime(match[3])
    const end = minuteOfDayToTime(match[7])
    if (!start || !end) return null
    if (match[4] === '&&' && start > end) return null
    if (match[4] === '||' && start <= end) return null

    timezone = match[1]
    ranges.push({ start, end })
  }

  if (!timezone || ranges.length === 0) return null
  return { variable: 'time_range', timezone, ranges }
}

export function timeRangeConditionPattern(): string {
  const minuteOfDay =
    'hour\\("[^"]+"\\)\\s*\\*\\s*60\\s*\\+\\s*minute\\("[^"]+"\\)'
  const segment = `\\(${minuteOfDay}\\s*>=\\s*\\d+\\s*(?:&&|\\|\\|)\\s*${minuteOfDay}\\s*<=\\s*\\d+\\)`
  return `(?:${segment}|\\(${segment}(?:\\s*\\|\\|\\s*${segment})+\\))`
}
