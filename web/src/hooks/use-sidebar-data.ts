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
import { LinkSquare01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import {
  Activity,
  Box,
  CreditCard,
  FileText,
  FlaskConical,
  Key,
  LayoutDashboard,
  ListTodo,
  MessageSquare,
  Radio,
  ReceiptText,
  ServerCog,
  Settings,
  Ticket,
  User,
  Users,
  Wallet,
} from 'lucide-react'
import { createElement, type ComponentProps } from 'react'
import { useTranslation } from 'react-i18next'

import type { SidebarData } from '@/components/layout/types'
import { useCustomMenus } from '@/hooks/use-custom-menus'
import { CUSTOM_MENU_LOCATION, CUSTOM_MENU_OPEN_MODE } from '@/lib/custom-menus'
import { ROLE } from '@/lib/roles'

function CustomMenuIcon(
  props: Omit<ComponentProps<typeof HugeiconsIcon>, 'icon'>
) {
  return createElement(HugeiconsIcon, {
    icon: LinkSquare01Icon,
    strokeWidth: 2,
    ...props,
  })
}

/**
 * Root navigation groups for the application sidebar.
 *
 * These are shown when the URL does not match any nested sidebar view
 * registered in `layout/lib/sidebar-view-registry.ts`.
 */
export function useSidebarData(): SidebarData {
  const { t } = useTranslation()
  const customMenuItems = useCustomMenus()
  const chatMenuItems = customMenuItems
    .filter((item) => item.location === CUSTOM_MENU_LOCATION.CHAT)
    .map((item) => ({
      title: item.name,
      url:
        item.openMode === CUSTOM_MENU_OPEN_MODE.EMBED
          ? `/custom-menu/${item.id}`
          : item.url,
      icon: CustomMenuIcon,
      external: item.openMode === CUSTOM_MENU_OPEN_MODE.NEW_TAB,
    }))
  const personalMenuItems = customMenuItems
    .filter((item) => item.location === CUSTOM_MENU_LOCATION.PERSONAL)
    .map((item) => ({
      title: item.name,
      url:
        item.openMode === CUSTOM_MENU_OPEN_MODE.EMBED
          ? `/custom-menu/${item.id}`
          : item.url,
      configUrls: [
        '/wallet',
        '/profile?view=consumption',
        '/profile?view=profile',
      ],
      icon: CustomMenuIcon,
      external: item.openMode === CUSTOM_MENU_OPEN_MODE.NEW_TAB,
    }))

  return {
    navGroups: [
      {
        id: 'chat',
        title: t('Chat'),
        items: [
          {
            title: t('Playground'),
            url: '/playground',
            icon: FlaskConical,
          },
          {
            title: t('Chat'),
            icon: MessageSquare,
            type: 'chat-presets',
          },
          ...chatMenuItems,
        ],
      },
      {
        id: 'general',
        title: t('General'),
        items: [
          {
            title: t('Overview'),
            url: '/dashboard/overview',
            icon: Activity,
          },
          {
            title: t('Dashboard'),
            url: '/dashboard/models',
            icon: LayoutDashboard,
          },
          {
            title: t('API Keys'),
            url: '/keys',
            icon: Key,
          },
          {
            title: t('Usage Logs'),
            url: '/usage-logs/common',
            icon: FileText,
          },
          {
            title: t('Task Logs'),
            url: '/usage-logs/task',
            activeUrls: ['/usage-logs/drawing'],
            configUrls: ['/usage-logs/drawing', '/usage-logs/task'],
            icon: ListTodo,
          },
        ],
      },
      {
        id: 'personal',
        title: t('Personal'),
        items: [
          {
            title: t('Wallet'),
            url: '/wallet',
            icon: Wallet,
          },
          {
            title: t('Consumption exports'),
            url: '/profile?view=consumption',
            icon: ReceiptText,
          },
          {
            title: t('Profile'),
            url: '/profile?view=profile',
            icon: User,
          },
          ...personalMenuItems,
        ],
      },
      {
        id: 'admin',
        title: t('Admin'),
        items: [
          {
            title: t('Channels'),
            url: '/channels',
            icon: Radio,
          },
          {
            title: t('Models'),
            url: '/models/metadata',
            icon: Box,
          },
          {
            title: t('Users'),
            url: '/users',
            icon: Users,
          },
          {
            title: t('Redemption Codes'),
            url: '/redemption-codes',
            icon: Ticket,
          },
          {
            title: t('Subscriptions'),
            url: '/subscriptions',
            icon: CreditCard,
          },
          {
            title: t('System Info'),
            url: '/system-info',
            icon: ServerCog,
            requiredRole: ROLE.SUPER_ADMIN,
          },
          {
            title: t('System Settings'),
            url: '/system-settings/site',
            activeUrls: ['/system-settings'],
            icon: Settings,
          },
        ],
      },
    ],
  }
}
