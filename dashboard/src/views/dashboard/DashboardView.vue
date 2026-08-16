<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Line, Doughnut } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  ArcElement,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js'
import { statsApi } from '@/api'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Users, UserCheck, UserPlus } from 'lucide-vue-next'

const { t } = useI18n()
ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, ArcElement, Tooltip, Legend, Filler)

const { data: stats, isLoading } = useQuery({
  queryKey: ['stats'],
  queryFn: statsApi.get,
  staleTime: 60_000,
})

const lineData = computed(() => ({
  labels: (stats.value?.signups_by_day ?? []).map((d) => {
    const dt = new Date(d.day)
    return dt.toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit' })
  }),
  datasets: [
    {
      label: t('dashboard.newUsersShort'),
      data: (stats.value?.signups_by_day ?? []).map((d) => d.count),
      borderColor: '#6366f1',
      backgroundColor: 'rgba(99, 102, 241, 0.15)',
      fill: true,
      tension: 0.4,
      pointRadius: 3,
    },
  ],
}))

const doughnutData = computed(() => ({
  labels: (stats.value?.role_distribution ?? []).map((r) => r.role),
  datasets: [
    {
      data: (stats.value?.role_distribution ?? []).map((r) => r.count),
      backgroundColor: ['#6366f1', '#8b5cf6', '#a855f7', '#60a5fa', '#d946ef'],
      borderWidth: 0,
    },
  ],
}))

const lineOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: { legend: { display: false } },
  scales: {
    x: { grid: { display: false }, ticks: { color: '#94a3b8', maxTicksLimit: 8 } },
    y: { grid: { color: 'rgba(148,163,184,0.12)' }, ticks: { color: '#94a3b8', precision: 0 } },
  },
}

const doughnutOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { position: 'bottom' as const, labels: { color: '#94a3b8', boxWidth: 12 } },
  },
}

const statCards = computed(() => [
  { label: t('dashboard.totalUsers'), value: stats.value?.total_users ?? 0, icon: Users },
  { label: t('dashboard.activeUsers'), value: stats.value?.active_users ?? 0, icon: UserCheck },
  { label: t('dashboard.recent30d'), value: stats.value?.recent_users ?? 0, icon: UserPlus },
])
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">{{ t('dashboard.title') }}</h1>
      <p class="text-sm text-muted-foreground">{{ t('dashboard.subtitle') }}</p>
    </div>

    <!-- Stat cards -->
    <div class="grid gap-4 sm:grid-cols-3">
      <Card v-for="card in statCards" :key="card.label">
        <CardContent class="flex items-center gap-4 p-6">
          <div class="flex h-11 w-11 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <component :is="card.icon" class="h-5 w-5" />
          </div>
          <div>
            <Skeleton v-if="isLoading" class="h-7 w-16" />
            <div v-else class="text-2xl font-bold">{{ card.value }}</div>
            <div class="text-xs text-muted-foreground">{{ card.label }}</div>
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- Charts -->
    <div class="grid gap-4 lg:grid-cols-3">
      <Card class="lg:col-span-2">
        <CardHeader>
          <CardTitle class="text-base">{{ t('dashboard.newUsers') }}</CardTitle>
          <CardDescription>{{ t('dashboard.registeredByDay') }}</CardDescription>
        </CardHeader>
        <CardContent>
          <div class="h-72">
            <Skeleton v-if="isLoading" class="h-full w-full" />
            <Line v-else :data="lineData" :options="lineOptions" />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle class="text-base">{{ t('dashboard.roleDistribution') }}</CardTitle>
          <CardDescription>{{ t('dashboard.byRole') }}</CardDescription>
        </CardHeader>
        <CardContent>
          <div class="h-72">
            <Skeleton v-if="isLoading" class="h-full w-full" />
            <Doughnut v-else :data="doughnutData" :options="doughnutOptions" />
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
