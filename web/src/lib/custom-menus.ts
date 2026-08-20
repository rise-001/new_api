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
import type { SystemStatus } from '@/features/auth/types'

export const CUSTOM_MENU_LOCATION = {
  CHAT: 'chat',
  PERSONAL: 'personal',
  TOP: 'top',
} as const

export const CUSTOM_MENU_OPEN_MODE = {
  EMBED: 'embed',
  NEW_TAB: 'new_tab',
} as const

export const CUSTOM_MENU_MAX_ITEMS = 50

export type CustomMenuLocation =
  (typeof CUSTOM_MENU_LOCATION)[keyof typeof CUSTOM_MENU_LOCATION]
export type CustomMenuOpenMode =
  (typeof CUSTOM_MENU_OPEN_MODE)[keyof typeof CUSTOM_MENU_OPEN_MODE]

export type CustomMenuItem = {
  id: string
  url: string
  name: string
  location: CustomMenuLocation
  openMode: CustomMenuOpenMode
}

type RawCustomMenuItem = {
  id?: unknown
  url?: unknown
  name?: unknown
  location?: unknown
  open_mode?: unknown
}

const CUSTOM_MENU_ID_PATTERN = /^[A-Za-z0-9_-]{1,64}$/

export function isHttpUrl(value: string): boolean {
  try {
    const parsed = new URL(value)
    return parsed.protocol === 'http:' || parsed.protocol === 'https:'
  } catch {
    return false
  }
}

export function parseCustomMenuItems(raw: unknown): CustomMenuItem[] {
  let parsed = raw
  if (typeof raw === 'string') {
    try {
      parsed = JSON.parse(raw)
    } catch {
      return []
    }
  }
  if (!Array.isArray(parsed)) return []

  return parsed.flatMap((value) => {
    if (!value || typeof value !== 'object' || Array.isArray(value)) return []
    const item = value as RawCustomMenuItem
    if (
      typeof item.id !== 'string' ||
      !CUSTOM_MENU_ID_PATTERN.test(item.id) ||
      typeof item.url !== 'string' ||
      !isHttpUrl(item.url) ||
      typeof item.name !== 'string' ||
      !item.name.trim()
    ) {
      return []
    }

    const location = item.location
    const openMode = item.open_mode
    const isSidebarItem =
      (location === CUSTOM_MENU_LOCATION.CHAT ||
        location === CUSTOM_MENU_LOCATION.PERSONAL) &&
      (openMode === CUSTOM_MENU_OPEN_MODE.EMBED ||
        openMode === CUSTOM_MENU_OPEN_MODE.NEW_TAB)
    const isTopItem =
      location === CUSTOM_MENU_LOCATION.TOP &&
      openMode === CUSTOM_MENU_OPEN_MODE.NEW_TAB
    if (!isSidebarItem && !isTopItem) return []

    return [
      {
        id: item.id,
        url: item.url,
        name: item.name,
        location,
        openMode,
      },
    ]
  })
}

export function parseCustomMenuItemsFromStatus(
  status: SystemStatus | null
): CustomMenuItem[] {
  const raw =
    status?.CustomMenuItems ??
    status?.data?.CustomMenuItems ??
    status?.custom_menu_items ??
    status?.data?.custom_menu_items
  return parseCustomMenuItems(raw)
}

export function serializeCustomMenuItems(items: CustomMenuItem[]): string {
  return JSON.stringify(
    items.map((item) => ({
      id: item.id,
      url: item.url.trim(),
      name: item.name.trim(),
      location: item.location,
      open_mode:
        item.location === CUSTOM_MENU_LOCATION.TOP
          ? CUSTOM_MENU_OPEN_MODE.NEW_TAB
          : item.openMode,
    }))
  )
}
