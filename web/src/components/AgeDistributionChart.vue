<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Chart from 'primevue/chart'
import { apiClient } from '@/api/client'
import type { AgeDistributionResponse } from '@/api/types'

const props = defineProps<{
  orgId: number | null
}>()

const { t } = useI18n()

const loading = ref(false)
const error = ref<string | null>(null)
const statsData = ref<AgeDistributionResponse | null>(null)

// Gender colors for stacked bars
const genderColors = {
  male: {
    background: 'rgba(59, 130, 246, 0.7)', // blue
    border: 'rgb(59, 130, 246)'
  },
  female: {
    background: 'rgba(236, 72, 153, 0.7)', // pink
    border: 'rgb(236, 72, 153)'
  },
  diverse: {
    background: 'rgba(168, 85, 247, 0.7)', // purple
    border: 'rgb(168, 85, 247)'
  }
}

const chartData = computed(() => {
  if (!statsData.value) {
    return { labels: [], datasets: [] }
  }

  const labels = statsData.value.distribution.map((bucket) =>
    bucket.age_label === '6+'
      ? t('statistics.ageSixPlus')
      : t('statistics.ageYears', { age: bucket.age_label })
  )

  const maleData = statsData.value.distribution.map((bucket) => bucket.male_count)
  const femaleData = statsData.value.distribution.map((bucket) => bucket.female_count)
  const diverseData = statsData.value.distribution.map((bucket) => bucket.diverse_count)

  return {
    labels,
    datasets: [
      {
        label: t('gender.male'),
        data: maleData,
        backgroundColor: genderColors.male.background,
        borderColor: genderColors.male.border,
        borderWidth: 1
      },
      {
        label: t('gender.female'),
        data: femaleData,
        backgroundColor: genderColors.female.background,
        borderColor: genderColors.female.border,
        borderWidth: 1
      },
      {
        label: t('gender.diverse'),
        data: diverseData,
        backgroundColor: genderColors.diverse.background,
        borderColor: genderColors.diverse.border,
        borderWidth: 1
      }
    ]
  }
})

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  layout: {
    padding: {
      top: 20
    }
  },
  plugins: {
    legend: {
      display: true,
      position: 'bottom' as const
    },
    datalabels: {
      display: false // Disable individual bar labels for stacked bars
    },
    tooltip: {
      callbacks: {
        footer: (items: { parsed: { y: number } }[]) => {
          const total = items.reduce((sum, item) => sum + item.parsed.y, 0)
          return `${t('common.total')}: ${total}`
        }
      }
    }
  },
  scales: {
    x: {
      stacked: true
    },
    y: {
      stacked: true,
      beginAtZero: true,
      grace: '15%',
      ticks: {
        stepSize: 1,
        precision: 0
      }
    }
  }
}))

async function loadStats() {
  if (!props.orgId) {
    statsData.value = null
    return
  }

  loading.value = true
  error.value = null

  try {
    statsData.value = await apiClient.getAgeDistribution(props.orgId)
  } catch (err) {
    console.error('Failed to load age distribution:', err)
    error.value = t('statistics.chartError')
  } finally {
    loading.value = false
  }
}

watch(
  () => props.orgId,
  () => {
    loadStats()
  }
)

onMounted(() => {
  loadStats()
})
</script>

<template>
  <div class="age-distribution-chart">
    <h3>{{ t('statistics.ageDistribution') }}</h3>

    <div v-if="loading" class="chart-loading">
      <i class="pi pi-spin pi-spinner"></i>
      {{ t('common.loading') }}
    </div>

    <div v-else-if="error" class="chart-error">
      <i class="pi pi-exclamation-triangle"></i>
      {{ error }}
    </div>

    <div v-else-if="!orgId" class="chart-placeholder">
      {{ t('statistics.selectOrgForStats') }}
    </div>

    <div v-else class="chart-wrapper">
      <div class="chart-header">
        <span class="total-count">
          {{ t('statistics.totalChildren', { count: statsData?.total_count || 0 }) }}
        </span>
        <span class="reference-date">
          {{ statsData?.date }}
        </span>
      </div>
      <div class="chart-container">
        <Chart type="bar" :data="chartData" :options="chartOptions" style="height: 300px" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.age-distribution-chart {
  background: var(--surface-card);
  border-radius: 8px;
  padding: 1.5rem;
}

.age-distribution-chart h3 {
  margin: 0 0 1rem;
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-color);
}

.chart-wrapper {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.5rem 0;
  border-bottom: 1px solid var(--surface-border);
}

.total-count {
  font-weight: 600;
  color: var(--text-color);
}

.reference-date {
  color: var(--text-color-secondary);
  font-size: 0.9rem;
}

.chart-container {
  height: 300px;
  min-height: 300px;
}

.chart-loading,
.chart-error,
.chart-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  height: 200px;
  color: var(--text-color-secondary);
}

.chart-error {
  color: var(--red-500);
}
</style>
