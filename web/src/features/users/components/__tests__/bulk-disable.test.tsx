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
import assert from 'node:assert/strict'

import { afterAll, afterEach, beforeEach, describe, test } from 'vitest'

import type { ApiResponse, User } from '../../types'

const { act, useState } = await import('react')
const { createRoot } = await import('react-dom/client')
const { QueryClient, QueryClientProvider, useQuery, notifyManager } =
  await import('@tanstack/react-query')
const { useReactTable, getCoreRowModel } = await import('@tanstack/react-table')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { toast } = await import('sonner')
const { AxiosError } = await import('axios')
const { api } = await import('@/lib/api')
const { useAuthStore } = await import('@/stores/auth-store')
const originalAuth = useAuthStore.getState().auth
const { DataTableBulkActions } = await import('../data-table-bulk-actions')
const { USER_ROLE, USER_STATUS } = await import('../../constants')

const i18n = createInstance()
await i18n.use(initReactI18next).init({ lng: 'en', resources: {} })
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
const originalActEnvironment = reactTestGlobals.IS_REACT_ACT_ENVIRONMENT
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true
notifyManager.setScheduler(queueMicrotask)

const originalAdapter = api.defaults.adapter
let serverUsers: User[]
let requests: { id: number; action: string }[]
let respond: (id: number) => Promise<ApiResponse>
let root: ReturnType<typeof createRoot> | undefined
let queryClient: InstanceType<typeof QueryClient>
let toastOffset: number

function user(id: number, overrides: Partial<User> = {}): User {
  return {
    id,
    username: `user-${id}`,
    display_name: `User ${id}`,
    quota: 0,
    used_quota: 0,
    request_count: 0,
    group: 'default',
    role: USER_ROLE.USER,
    status: USER_STATUS.ENABLED,
    ...overrides,
  }
}

function UsersSelection(props: { selectedIds: number[] }) {
  const { data = [] } = useQuery({
    queryKey: ['users', 'bulk-disable-test'],
    queryFn: async () => serverUsers,
    initialData: serverUsers,
    staleTime: Infinity,
  })
  const [rowSelection, setRowSelection] = useState(
    Object.fromEntries(props.selectedIds.map((id) => [String(id), true]))
  )
  const table = useReactTable({
    data,
    columns: [],
    getCoreRowModel: getCoreRowModel(),
    getRowId: (row) => String(row.id),
    state: { rowSelection },
    onRowSelectionChange: setRowSelection,
  })
  return (
    <>
      {table.getRowModel().rows.map((row) => (
        <label key={row.id}>
          <input
            type='checkbox'
            aria-label={row.original.username}
            checked={row.getIsSelected()}
            onChange={row.getToggleSelectedHandler()}
          />
          {row.original.username}
        </label>
      ))}
      <DataTableBulkActions
        table={table}
        requestVerification={async (request) => {
          assert.equal(request.scope, 'admin.user.manage')
          assert.ok(request.context && 'user_id' in request.context)
          return {
            proof_token: `proof-${request.context.user_id}`,
            scope: request.scope,
            method: 'passkey',
            expires_at: 2000000000,
          }
        }}
      />
    </>
  )
}

async function renderUsers(
  users: User[],
  selectedIds = users.map((item) => item.id)
) {
  serverUsers = users
  const container = document.createElement('div')
  document.body.append(container)
  root = createRoot(container)
  await act(async () => {
    root?.render(
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>
          <UsersSelection selectedIds={selectedIds} />
        </I18nextProvider>
      </QueryClientProvider>
    )
  })
}

async function confirmDisable() {
  const completed = new Promise<void>((resolve) => {
    const unsubscribe = queryClient.getMutationCache().subscribe((event) => {
      if (
        event.type === 'updated' &&
        event.mutation.state.status === 'success'
      ) {
        unsubscribe()
        resolve()
      }
    })
  })
  const confirm = [
    ...document.querySelectorAll<HTMLButtonElement>(
      '[role="alertdialog"] button'
    ),
  ].find((button) => button.textContent === 'Disable')
  assert.ok(confirm)
  await act(async () => {
    confirm.click()
    await completed
  })
}

describe('bulk user disable', () => {
  beforeEach(() => {
    useAuthStore.setState({
      auth: {
        ...originalAuth,
        accessToken: 'test-access',
        accessExpiresAt: Math.floor(Date.now() / 1000) + 3600,
      },
    })
    requests = []
    respond = async () => ({ success: true })
    toastOffset = toast.getHistory().length
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    })
    api.defaults.adapter = async (config) => {
      assert.equal(config.url, '/api/user/manage')
      const payload = JSON.parse(config.data as string) as {
        id: number
        action: string
      }
      requests.push(payload)
      assert.equal(
        config.headers.get('X-Security-Proof'),
        `proof-${payload.id}`
      )
      let data: ApiResponse
      try {
        data = await respond(payload.id)
      } catch {
        throw new AxiosError('Network unavailable', 'ERR_NETWORK', config)
      }
      if (data.success) {
        serverUsers = serverUsers.map((item) =>
          item.id === payload.id
            ? { ...item, status: USER_STATUS.DISABLED }
            : item
        )
      }
      return { config, data, status: 200, statusText: 'OK', headers: {} }
    }
  })

  afterEach(async () => {
    useAuthStore.setState({ auth: originalAuth })
    await act(async () => root?.unmount())
    root = undefined
    queryClient.clear()
    api.defaults.adapter = originalAdapter
    toast.dismiss()
    document.body.replaceChildren()
  })

  afterAll(() => {
    notifyManager.setScheduler((callback) => setTimeout(callback, 0))
    reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = originalActEnvironment
  })

  test('no selection hides the action toolbar', async () => {
    await renderUsers([user(1)], [])
    assert.equal(document.querySelector('[role="toolbar"]'), null)
  })

  test('only disabled, deleted, or root users selected disables the action', async () => {
    await renderUsers([
      user(1, { status: USER_STATUS.DISABLED }),
      user(2, { DeletedAt: '2026-11-17T00:00:00Z' }),
      user(3, { role: USER_ROLE.ROOT }),
    ])
    const button = document.querySelector<HTMLButtonElement>(
      '[aria-label="Disable selected users"]'
    )
    assert.ok(button)
    assert.equal(button.disabled, true)
    assert.deepEqual(requests, [])
  })

  test('canceling confirmation leaves the selected user enabled', async () => {
    await renderUsers([user(1)])
    await act(async () =>
      document
        .querySelector<HTMLButtonElement>(
          '[aria-label="Disable selected users"]'
        )
        ?.click()
    )
    assert.ok(document.querySelector('[role="alertdialog"]'))
    assert.deepEqual(requests, [])
    const cancel = [
      ...document.querySelectorAll<HTMLButtonElement>(
        '[role="alertdialog"] button'
      ),
    ].find((button) => button.textContent === 'Cancel')
    assert.ok(cancel)
    await act(async () => cancel.click())
    assert.deepEqual(requests, [])
    assert.equal(
      document.querySelector<HTMLInputElement>('[aria-label="user-1"]')
        ?.checked,
      true
    )
  })

  test('End key in the toolbar focuses the disable action', async () => {
    await renderUsers([user(1)])
    const clear = document.querySelector<HTMLButtonElement>(
      '[aria-label="Clear selection"]'
    )
    const disable = document.querySelector<HTMLButtonElement>(
      '[aria-label="Disable selected users"]'
    )
    assert.ok(clear)
    assert.ok(disable)
    await act(async () => {
      clear.focus()
      clear.dispatchEvent(
        new KeyboardEvent('keydown', {
          key: 'End',
          bubbles: true,
        }) as unknown as Event
      )
    })
    assert.equal(document.activeElement, disable)
  })

  test('confirmation disables selected eligible users and refreshes their statuses', async () => {
    await renderUsers(
      [
        user(11),
        user(12),
        user(13),
        user(14, { role: USER_ROLE.ROOT }),
        user(15, { status: USER_STATUS.DISABLED }),
        user(16, { DeletedAt: '2026-11-17T00:00:00Z' }),
      ],
      [11, 12, 14, 15, 16]
    )
    await act(async () =>
      document
        .querySelector<HTMLButtonElement>(
          '[aria-label="Disable selected users"]'
        )
        ?.click()
    )
    assert.match(
      document.querySelector('[role="alertdialog"]')?.textContent ?? '',
      /Disable 2 selected user/
    )
    await confirmDisable()
    assert.deepEqual(requests, [
      { id: 11, action: 'disable' },
      { id: 12, action: 'disable' },
    ])
    const refreshed = queryClient.getQueryData<User[]>([
      'users',
      'bulk-disable-test',
    ])
    assert.equal(
      refreshed?.find((item) => item.id === 11)?.status,
      USER_STATUS.DISABLED
    )
    assert.equal(
      refreshed?.find((item) => item.id === 12)?.status,
      USER_STATUS.DISABLED
    )
    assert.equal(
      document.querySelector<HTMLInputElement>('[aria-label="user-11"]')
        ?.checked,
      false
    )
    assert.equal(
      document.querySelector<HTMLInputElement>('[aria-label="user-12"]')
        ?.checked,
      false
    )
    const notification = toast.getHistory().slice(toastOffset)[0]
    assert.ok(notification && 'title' in notification)
    assert.equal(notification.title, 'Disabled 2 user(s)')
  })

  test('business and network failures retain only failed selections after reordered refresh', async () => {
    respond = async (id) => {
      if (id === 12) return { success: false, message: 'Permission denied' }
      if (id === 13) throw new Error('Network unavailable')
      serverUsers = [...serverUsers].reverse()
      return { success: true }
    }
    await renderUsers([user(11), user(12), user(13)])
    await act(async () =>
      document
        .querySelector<HTMLButtonElement>(
          '[aria-label="Disable selected users"]'
        )
        ?.click()
    )
    await confirmDisable()
    assert.equal(
      document.querySelector<HTMLInputElement>('[aria-label="user-11"]')
        ?.checked,
      false
    )
    assert.equal(
      document.querySelector<HTMLInputElement>('[aria-label="user-12"]')
        ?.checked,
      true
    )
    assert.equal(
      document.querySelector<HTMLInputElement>('[aria-label="user-13"]')
        ?.checked,
      true
    )
    assert.deepEqual(
      toast
        .getHistory()
        .slice(toastOffset)
        .map((item) => ('title' in item ? item.title : undefined)),
      ['Disabled 1 user(s)', 'Failed to disable 2 user(s). Please try again.']
    )
  })

  test('all requests failing preserves the selection for retry without a success message', async () => {
    respond = async () => ({ success: false, message: 'Permission denied' })
    await renderUsers([user(11)])
    await act(async () =>
      document
        .querySelector<HTMLButtonElement>(
          '[aria-label="Disable selected users"]'
        )
        ?.click()
    )
    await confirmDisable()
    assert.equal(
      document.querySelector<HTMLInputElement>('[aria-label="user-11"]')
        ?.checked,
      true
    )
    assert.equal(
      document.querySelector<HTMLButtonElement>(
        '[aria-label="Disable selected users"]'
      )?.disabled,
      false
    )
    assert.deepEqual(
      toast
        .getHistory()
        .slice(toastOffset)
        .map((item) => ('title' in item ? item.title : undefined)),
      ['Failed to disable 1 user(s). Please try again.']
    )
  })

  test('pending requests prevent duplicate confirmation and canceling', async () => {
    let finish: (result: ApiResponse) => void = () => undefined
    respond = () =>
      new Promise((resolve) => {
        finish = resolve
      })
    await renderUsers([user(11)])
    await act(async () =>
      document
        .querySelector<HTMLButtonElement>(
          '[aria-label="Disable selected users"]'
        )
        ?.click()
    )
    const confirm = [
      ...document.querySelectorAll<HTMLButtonElement>(
        '[role="alertdialog"] button'
      ),
    ].find((button) => button.textContent === 'Disable')
    assert.ok(confirm)
    await act(async () => confirm.click())
    assert.equal(confirm.disabled, true)
    assert.equal(confirm.textContent, 'Disabling...')
    const cancel = [
      ...document.querySelectorAll<HTMLButtonElement>(
        '[role="alertdialog"] button'
      ),
    ].find((button) => button.textContent === 'Cancel')
    assert.ok(cancel)
    assert.equal(cancel.disabled, true)
    await act(async () => confirm.click())
    assert.equal(requests.length, 1)
    await act(async () => finish({ success: true }))
  })
})
