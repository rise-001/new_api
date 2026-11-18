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

import { test } from 'vitest'

import type { ApiKey } from '../../types'
import {
  getApiKeyFormDefaultValues,
  transformApiKeyToFormDefaults,
  transformFormDataToPayload,
} from '../api-key-form'

test('switching a new key to auto enables cross-group retry without a toggle', () => {
  const form = getApiKeyFormDefaultValues(false)
  form.group = 'auto'

  assert.equal(transformFormDataToPayload(form).cross_group_retry, true)
})

test('saving a legacy auto key enables cross-group retry even if it was disabled', () => {
  const key: ApiKey = {
    id: 1,
    name: 'legacy-auto-key',
    key: 'masked-key',
    status: 1,
    remain_quota: 0,
    used_quota: 0,
    unlimited_quota: true,
    expired_time: -1,
    created_time: 1,
    accessed_time: 1,
    group: 'auto',
    auto_groups: null,
    allow_user_quota_query: false,
    cross_group_retry: false,
    model_limits_enabled: false,
    model_limits: '',
    allow_ips: '',
  }

  const payload = transformFormDataToPayload(transformApiKeyToFormDefaults(key))

  assert.equal(payload.cross_group_retry, true)
  assert.equal(payload.group, 'auto')
})

test('switching from auto to a fixed group keeps routing within the selected group', () => {
  const form = getApiKeyFormDefaultValues(true)
  form.group = 'default'

  assert.equal(transformFormDataToPayload(form).cross_group_retry, false)
})
