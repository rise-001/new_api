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
import { after, afterEach, describe, test } from 'node:test'

import { Window } from 'happy-dom'

import type { PricingModel } from '../../types'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'CustomEvent',
  'customElements',
  'MutationObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
  'matchMedia',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { Tabs } = await import('@/components/ui/tabs')
const { ROLE } = await import('@/lib/roles')
const { useAuthStore } = await import('@/stores/auth-store')
const { ModelCardGrid } = await import('../model-card-grid')
const { ModelDetailsTabList } = await import('../model-details-tab-list')
const { ModelOverviewSummary } = await import('../model-overview-summary')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'Average latency': 'Average latency',
        API: 'API',
        'Latency short': 'Latency',
        Overview: 'Overview',
        Performance: 'Performance',
        'Status short': 'Status',
        'Success rate': 'Success rate',
        'Throughput short': 'Throughput',
      },
    },
  },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const model: PricingModel = {
  id: 1,
  model_name: 'test-model',
  quota_type: 0,
  model_ratio: 1,
  completion_ratio: 1,
  enable_groups: ['default'],
}

function buildStatus(overviewMetricsAdminOnly: boolean) {
  return {
    HeaderNavModules: JSON.stringify({
      pricing: {
        enabled: true,
        requireAuth: false,
        overviewMetricsAdminOnly,
      },
    }),
  }
}

async function renderSummary(options: {
  overviewMetricsAdminOnly: boolean
  role: number
  withMetrics?: boolean
}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  queryClient.setQueryData(
    ['status'],
    buildStatus(options.overviewMetricsAdminOnly)
  )

  if (options.withMetrics) {
    queryClient.setQueryData(['perf-metrics', model.model_name], {
      success: true,
      data: {
        model_name: model.model_name,
        groups: [
          {
            group: 'default',
            avg_ttft_ms: 100,
            avg_latency_ms: 800,
            success_rate: 99,
            avg_tps: 25,
            series: [],
          },
        ],
      },
    })
  }

  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'viewer',
    role: options.role,
  })

  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>
          <ModelOverviewSummary model={model} />
        </I18nextProvider>
      </QueryClientProvider>
    )
  })

  return { container, queryClient, root }
}

async function renderCardGrid(options: {
  overviewMetricsAdminOnly: boolean
  role: number
}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  queryClient.setQueryData(
    ['status'],
    buildStatus(options.overviewMetricsAdminOnly)
  )
  queryClient.setQueryData(['perf-metrics-summary', 24], {
    success: true,
    data: {
      models: [
        {
          model_name: model.model_name,
          avg_latency_ms: 800,
          success_rate: 99,
          avg_tps: 25,
          recent_success_rates: [99, 99, 99],
        },
      ],
    },
  })

  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'viewer',
    role: options.role,
  })

  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>
          <ModelCardGrid models={[model]} onModelClick={() => undefined} />
        </I18nextProvider>
      </QueryClientProvider>
    )
  })

  return { container, queryClient, root }
}

async function cleanupRendered(
  rendered: Awaited<ReturnType<typeof renderSummary>>
) {
  await act(async () => rendered.root.unmount())
  rendered.container.remove()
  rendered.queryClient.clear()
}

async function renderTabList(showPerformance: boolean) {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(
      <I18nextProvider i18n={i18n}>
        <Tabs defaultValue='overview'>
          <ModelDetailsTabList showPerformance={showPerformance} />
        </Tabs>
      </I18nextProvider>
    )
  })

  return { container, root }
}

async function cleanupTabList(
  rendered: Awaited<ReturnType<typeof renderTabList>>
) {
  await act(async () => rendered.root.unmount())
  rendered.container.remove()
}

describe('model performance visibility', () => {
  afterEach(() => {
    useAuthStore.getState().auth.reset()
  })

  after(() => {
    domWindow.close()
  })

  test('hides overview metrics and skips their query for a regular user', async () => {
    const rendered = await renderSummary({
      overviewMetricsAdminOnly: true,
      role: ROLE.USER,
    })

    assert.equal(rendered.container.textContent, '')
    assert.equal(
      rendered.queryClient.getQueryState(['perf-metrics', model.model_name]),
      undefined
    )

    await cleanupRendered(rendered)
  })

  test('shows overview metrics to an administrator', async () => {
    const rendered = await renderSummary({
      overviewMetricsAdminOnly: true,
      role: ROLE.ADMIN,
      withMetrics: true,
    })

    assert.match(rendered.container.textContent ?? '', /TPS/)
    assert.match(rendered.container.textContent ?? '', /Average latency/)
    assert.match(rendered.container.textContent ?? '', /Success rate/)

    await cleanupRendered(rendered)
  })

  test('keeps overview metrics visible to a regular user when disabled', async () => {
    const rendered = await renderSummary({
      overviewMetricsAdminOnly: false,
      role: ROLE.USER,
      withMetrics: true,
    })

    assert.match(rendered.container.textContent ?? '', /TPS/)

    await cleanupRendered(rendered)
  })

  test('hides card performance metrics from a regular user', async () => {
    const rendered = await renderCardGrid({
      overviewMetricsAdminOnly: true,
      role: ROLE.USER,
    })

    assert.doesNotMatch(
      rendered.container.textContent ?? '',
      /Latency|Throughput|Status/
    )

    await cleanupRendered(rendered)
  })

  test('keeps card performance metrics visible to an administrator', async () => {
    const rendered = await renderCardGrid({
      overviewMetricsAdminOnly: true,
      role: ROLE.ADMIN,
    })

    assert.match(rendered.container.textContent ?? '', /Latency/)
    assert.match(rendered.container.textContent ?? '', /Throughput/)
    assert.match(rendered.container.textContent ?? '', /Status/)

    await cleanupRendered(rendered)
  })

  test('hides the Performance tab from a regular user', async () => {
    const rendered = await renderTabList(false)
    const tabs = [...rendered.container.querySelectorAll('[role="tab"]')].map(
      (tab) => tab.textContent
    )

    assert.deepEqual(tabs, ['Overview', 'API'])

    await cleanupTabList(rendered)
  })

  test('keeps the Performance tab visible to an administrator', async () => {
    const rendered = await renderTabList(true)
    const tabs = [...rendered.container.querySelectorAll('[role="tab"]')].map(
      (tab) => tab.textContent
    )

    assert.deepEqual(tabs, ['Overview', 'Performance', 'API'])

    await cleanupTabList(rendered)
  })
})
