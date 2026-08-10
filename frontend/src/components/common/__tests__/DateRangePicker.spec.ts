import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { ref } from 'vue'

import DateRangePicker from '../DateRangePicker.vue'

const messages: Record<string, string> = {
  'dates.today': 'Today',
  'dates.yesterday': 'Yesterday',
  'dates.last24Hours': 'Last 24 Hours',
  'dates.last7Days': 'Last 7 Days',
  'dates.last14Days': 'Last 14 Days',
  'dates.last30Days': 'Last 30 Days',
  'dates.thisMonth': 'This Month',
  'dates.lastMonth': 'Last Month',
  'dates.startDate': 'Start Date',
  'dates.endDate': 'End Date',
  'dates.apply': 'Apply',
  'dates.selectDateRange': 'Select date range'
}

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => messages[key] ?? key,
    locale: ref('en')
  })
}))

const formatLocalDate = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

describe('DateRangePicker', () => {
  it('uses last 24 hours as the default recognized preset', () => {
    const now = new Date()
    const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000)

    const wrapper = mount(DateRangePicker, {
      props: {
        startDate: formatLocalDate(yesterday),
        endDate: formatLocalDate(now)
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    expect(wrapper.text()).toContain('Last 24 Hours')
  })

  it('emits range updates with last24Hours preset when applied', async () => {
    const now = new Date()
    const today = formatLocalDate(now)

    const wrapper = mount(DateRangePicker, {
      props: {
        startDate: today,
        endDate: today
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    await wrapper.find('.date-picker-trigger').trigger('click')
    const presetButton = wrapper.findAll('.date-picker-preset').find((node) =>
      node.text().includes('Last 24 Hours')
    )
    expect(presetButton).toBeDefined()

    await presetButton!.trigger('click')
    await wrapper.find('.date-picker-apply').trigger('click')

    const change = wrapper.emitted('change')?.[0]?.[0] as {
      startDate: string
      endDate: string
      startTime?: string
      endTime?: string
      preset: string | null
    }

    expect(change.preset).toBe('last24Hours')
    expect(change.startTime).toBeTruthy()
    expect(change.endTime).toBeTruthy()

    const startMs = Date.parse(change.startTime!)
    const endMs = Date.parse(change.endTime!)
    expect(Number.isFinite(startMs)).toBe(true)
    expect(Number.isFinite(endMs)).toBe(true)
    // Rolling 24h window (±2s for test execution skew)
    expect(Math.abs(endMs - startMs - 24 * 60 * 60 * 1000)).toBeLessThanOrEqual(2000)
    expect(Math.abs(endMs - Date.now())).toBeLessThanOrEqual(5000)

    expect(wrapper.emitted('update:startDate')?.at(-1)).toEqual([change.startDate])
    expect(wrapper.emitted('update:endDate')?.at(-1)).toEqual([change.endDate])
  })

  it('does not emit precise times for day-level presets', async () => {
    const now = new Date()
    const today = formatLocalDate(now)

    const wrapper = mount(DateRangePicker, {
      props: {
        startDate: today,
        endDate: today
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    await wrapper.find('.date-picker-trigger').trigger('click')
    const presetButton = wrapper.findAll('.date-picker-preset').find((node) =>
      node.text().includes('Today')
    )
    expect(presetButton).toBeDefined()
    await presetButton!.trigger('click')
    await wrapper.find('.date-picker-apply').trigger('click')

    const change = wrapper.emitted('change')?.[0]?.[0] as {
      startTime?: string
      endTime?: string
      preset: string | null
    }
    expect(change.preset).toBe('today')
    expect(change.startTime).toBeUndefined()
    expect(change.endTime).toBeUndefined()
  })
})
