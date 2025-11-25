<template>
  <div class="schedule-settings">
    <!-- Loading State -->
    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>Loading schedule configuration...</p>
    </div>

    <!-- Main Content -->
    <div v-else class="settings-container">
      <!-- Header Section -->
      <div class="settings-header">
        <div class="header-content">
          <h3>Automated Health Checks</h3>
          <p>Run health checks automatically on a schedule and get notified via Slack</p>
        </div>
        <div class="header-toggle">
          <label class="toggle-switch">
            <input type="checkbox" v-model="formData.enabled" />
            <span class="slider"></span>
          </label>
          <span class="toggle-label">{{ formData.enabled ? 'Enabled' : 'Disabled' }}</span>
        </div>
      </div>

      <!-- Schedule Configuration -->
      <div class="config-section">
        <div class="section-icon">⏰</div>
        <div class="section-content">
          <h4>Schedule Frequency</h4>
          <p class="section-description">How often should health checks run?</p>
          
          <div class="frequency-selector">
            <label 
              v-for="interval in intervalOptions" 
              :key="interval.value"
              class="frequency-option"
              :class="{ active: scheduleInterval === interval.value }"
            >
              <input 
                type="radio" 
                :value="interval.value" 
                v-model="scheduleInterval"
                name="interval"
              />
              <div class="option-content">
                <span class="option-label">{{ interval.label }}</span>
                <span class="option-description">{{ interval.description }}</span>
              </div>
            </label>
          </div>

          <div class="cron-display">
            <code>{{ cronExpression }}</code>
            <span class="cron-label">Cron expression</span>
          </div>
        </div>
      </div>

      <!-- Slack Notifications -->
      <div class="config-section">
        <div class="section-icon">📢</div>
        <div class="section-content">
          <h4>Slack Notifications</h4>
          <p class="section-description">Get instant alerts when issues are detected</p>

          <div class="form-grid">
            <div class="form-field full-width">
              <label>Webhook URL <span class="required">*</span></label>
              <input 
                v-model="formData.webhookUrl" 
                type="url" 
                placeholder="https://hooks.slack.com/services/YOUR/WEBHOOK/URL" 
                class="input-field"
              />
              <span class="field-hint">
                Don't have a webhook? 
                <a href="https://api.slack.com/messaging/webhooks" target="_blank" rel="noopener">
                  Create one here
                </a>
              </span>
            </div>

            <div class="form-field">
              <label>Channel (optional)</label>
              <input 
                v-model="formData.channel" 
                type="text" 
                placeholder="#monitoring" 
                class="input-field"
              />
              <span class="field-hint">Override default channel</span>
            </div>

            <div class="form-field">
              <label class="checkbox-field">
                <input type="checkbox" v-model="formData.onlyOnFailure" />
                <span class="checkbox-label">
                  <strong>Only on failures</strong>
                  <span class="checkbox-description">Don't notify for successful checks</span>
                </span>
              </label>
            </div>
          </div>

          <!-- Test Notification -->
          <div v-if="formData.webhookUrl" class="test-notification">
            <button 
              @click="testNotification" 
              :disabled="testingNotification"
              class="btn btn-outline"
            >
              <span v-if="testingNotification">Sending...</span>
              <span v-else>🔔 Send Test Notification</span>
            </button>
            <span class="test-hint">Verify your Slack integration works</span>
          </div>
        </div>
      </div>

      <!-- Success/Error Messages -->
      <div v-if="successMessage" class="message message-success">
        <span class="message-icon">✓</span>
        {{ successMessage }}
      </div>
      <div v-if="errorMessage" class="message message-error">
        <span class="message-icon">⚠</span>
        {{ errorMessage }}
      </div>

      <!-- Action Buttons -->
      <div class="action-buttons">
        <button 
          v-if="hasSchedule" 
          @click="deleteSchedule" 
          :disabled="deleting"
          class="btn btn-danger-outline"
        >
          {{ deleting ? 'Deleting...' : 'Delete Schedule' }}
        </button>
        <button 
          @click="saveSchedule" 
          :disabled="saving || !formData.webhookUrl" 
          class="btn btn-primary"
        >
          {{ saving ? 'Saving...' : hasSchedule ? 'Update Schedule' : 'Create Schedule' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useScheduleStore } from '@/stores/schedule'
import type { NotificationConfig } from '@/types'

const props = defineProps<{
  workflowId: string
}>()

const scheduleStore = useScheduleStore()

const loading = ref(false)
const saving = ref(false)
const deleting = ref(false)
const testingNotification = ref(false)
const successMessage = ref('')
const errorMessage = ref('')
const scheduleInterval = ref('6')

const intervalOptions = [
  { value: '1', label: 'Every Hour', description: 'Run 24 times/day' },
  { value: '6', label: 'Every 6 Hours', description: 'Run 4 times/day' },
  { value: '12', label: 'Every 12 Hours', description: 'Run 2 times/day' },
  { value: '24', label: 'Daily', description: 'Run once/day' }
]

const formData = ref({
  enabled: true,
  webhookUrl: '',
  channel: '',
  onlyOnFailure: true
})

const hasSchedule = computed(() => {
  return scheduleStore.getSchedule(props.workflowId) !== undefined
})

const cronExpression = computed(() => {
  return `0 */${scheduleInterval.value} * * *`
})

onMounted(async () => {
  loading.value = true
  try {
    const schedule = await scheduleStore.fetchSchedule(props.workflowId)
    if (schedule) {
      formData.value.enabled = schedule.enabled
      formData.value.webhookUrl = schedule.notification_config?.slack?.webhook_url || ''
      formData.value.channel = schedule.notification_config?.slack?.channel || ''
      formData.value.onlyOnFailure = schedule.notification_config?.only_on_failure ?? true
    }
  } finally {
    loading.value = false
  }
})

async function saveSchedule() {
  saving.value = true
  successMessage.value = ''
  errorMessage.value = ''
  
  try {
    const notificationConfig: NotificationConfig = {
      only_on_failure: formData.value.onlyOnFailure
    }
    
    if (formData.value.webhookUrl) {
      notificationConfig.slack = {
        webhook_url: formData.value.webhookUrl,
        channel: formData.value.channel || undefined
      }
    }

    await scheduleStore.saveSchedule(props.workflowId, {
      schedule: cronExpression.value,
      enabled: formData.value.enabled,
      notification_config: notificationConfig
    })
    
    successMessage.value = 'Schedule saved successfully! Health checks will run automatically.'
    setTimeout(() => successMessage.value = '', 5000)
  } catch (error) {
    errorMessage.value = 'Failed to save schedule. Please try again.'
  } finally {
    saving.value = false
  }
}

async function deleteSchedule() {
  if (!confirm('Are you sure you want to delete this schedule? Automated health checks will stop.')) return
  
  deleting.value = true
  successMessage.value = ''
  errorMessage.value = ''
  
  try {
    await scheduleStore.deleteSchedule(props.workflowId)
    successMessage.value = 'Schedule deleted successfully.'
    formData.value = {
      enabled: true,
      webhookUrl: '',
      channel: '',
      onlyOnFailure: true
    }
    setTimeout(() => successMessage.value = '', 3000)
  } catch (error) {
    errorMessage.value = 'Failed to delete schedule.'
  } finally {
    deleting.value = false
  }
}

async function testNotification() {
  testingNotification.value = true
  successMessage.value = ''
  errorMessage.value = ''
  
  try {
    const config: NotificationConfig = {
      slack: {
        webhook_url: formData.value.webhookUrl,
        channel: formData.value.channel || undefined
      },
      only_on_failure: false
    }
    
    await scheduleStore.testNotification(props.workflowId, config)
    successMessage.value = '✓ Test notification sent! Check your Slack channel.'
    setTimeout(() => successMessage.value = '', 5000)
  } catch (error) {
    errorMessage.value = 'Failed to send test notification. Check your webhook URL.'
  } finally {
    testingNotification.value = false
  }
}
</script>

<style scoped>
.schedule-settings {
  max-width: 100%;
}

/* Loading State */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 2rem 1rem;
  text-align: center;
}

.spinner {
  width: 32px;
  height: 32px;
  border: 2px solid hsl(var(--border));
  border-top-color: hsl(var(--primary));
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
  margin-bottom: 0.75rem;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Settings Container */
.settings-container {
  background: transparent;
}

/* Header Section */
.settings-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 0 0 1.25rem 0;
  border-bottom: 1px solid hsl(var(--border));
}

.header-content h3 {
  margin: 0 0 0.25rem 0;
  font-size: 0.875rem;
  font-weight: 600;
  color: hsl(var(--foreground));
}

.header-content p {
  margin: 0;
  color: hsl(var(--muted-foreground));
  font-size: 0.75rem;
  line-height: 1.4;
}

.header-toggle {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
}

.toggle-switch {
  position: relative;
  display: inline-block;
  width: 40px;
  height: 22px;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: hsl(var(--muted));
  transition: 0.2s;
  border-radius: 11px;
}

.slider:before {
  position: absolute;
  content: "";
  height: 16px;
  width: 16px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  transition: 0.2s;
  border-radius: 50%;
}

.toggle-switch input:checked + .slider {
  background-color: hsl(var(--primary));
}

.toggle-switch input:checked + .slider:before {
  transform: translateX(18px);
}

.toggle-label {
  font-weight: 500;
  font-size: 0.75rem;
  color: hsl(var(--muted-foreground));
}

/* Config Sections */
.config-section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1.25rem 0;
  border-bottom: 1px solid hsl(var(--border));
}

.config-section:last-of-type {
  border-bottom: none;
  padding-bottom: 0;
}

.section-icon {
  display: none;
}

.section-content {
  flex: 1;
}

.section-content h4 {
  margin: 0 0 0.25rem 0;
  font-size: 0.8125rem;
  font-weight: 600;
  color: hsl(var(--foreground));
}

.section-description {
  margin: 0 0 1rem 0;
  color: hsl(var(--muted-foreground));
  font-size: 0.75rem;
  line-height: 1.4;
}

/* Frequency Selector */
.frequency-selector {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.frequency-option {
  position: relative;
  cursor: pointer;
}

.frequency-option input {
  position: absolute;
  opacity: 0;
}

.option-content {
  display: flex;
  flex-direction: column;
  padding: 0.625rem 0.75rem;
  border: 1px solid hsl(var(--border));
  border-radius: 6px;
  transition: all 0.15s;
  background: hsl(var(--background));
}

.frequency-option.active .option-content {
  border-color: hsl(var(--primary));
  background: hsl(var(--primary) / 0.04);
}

.option-label {
  font-weight: 500;
  font-size: 0.8125rem;
  color: hsl(var(--foreground));
  margin-bottom: 0.125rem;
}

.option-description {
  font-size: 0.6875rem;
  color: hsl(var(--muted-foreground));
}

.cron-display {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  background: hsl(var(--muted) / 0.3);
  border-radius: 4px;
  border: 1px solid hsl(var(--border));
}

.cron-display code {
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 0.75rem;
  color: hsl(var(--foreground));
  font-weight: 500;
}

.cron-label {
  font-size: 0.6875rem;
  color: hsl(var(--muted-foreground));
}

/* Form Grid */
.form-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1rem;
}

.form-field.full-width {
  grid-column: 1 / -1;
}

.form-field label {
  display: block;
  margin-bottom: 0.375rem;
  font-weight: 500;
  font-size: 0.75rem;
  color: hsl(var(--foreground));
}

.required {
  color: hsl(var(--destructive));
}

.input-field {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid hsl(var(--border));
  border-radius: 6px;
  font-size: 0.8125rem;
  transition: all 0.15s;
  background: hsl(var(--background));
  color: hsl(var(--foreground));
}

.input-field:focus {
  outline: none;
  border-color: hsl(var(--primary));
  box-shadow: 0 0 0 2px hsl(var(--primary) / 0.08);
}

.field-hint {
  display: block;
  margin-top: 0.375rem;
  font-size: 0.6875rem;
  color: hsl(var(--muted-foreground));
  line-height: 1.4;
}

.field-hint a {
  color: hsl(var(--primary));
  text-decoration: none;
}

.field-hint a:hover {
  text-decoration: underline;
}

.checkbox-field {
  display: flex;
  align-items: flex-start;
  gap: 0.625rem;
  cursor: pointer;
  padding: 0.75rem;
  border: 1px solid hsl(var(--border));
  border-radius: 6px;
  transition: all 0.15s;
}

.checkbox-field:hover {
  background: hsl(var(--muted) / 0.2);
}

.checkbox-field input {
  margin-top: 0.125rem;
}

.checkbox-label {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.checkbox-label strong {
  font-size: 0.8125rem;
  font-weight: 500;
}

.checkbox-description {
  font-size: 0.6875rem;
  color: hsl(var(--muted-foreground));
  font-weight: normal;
  line-height: 1.4;
}

/* Test Notification */
.test-notification {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid hsl(var(--border));
}

.test-hint {
  font-size: 0.6875rem;
  color: hsl(var(--muted-foreground));
}

/* Messages */
.message {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  padding: 0.75rem 1rem;
  margin: 1rem 0;
  border-radius: 6px;
  font-size: 0.75rem;
}

.message-icon {
  font-size: 1rem;
  font-weight: bold;
}

.message-success {
  background: hsl(142 76% 36% / 0.08);
  color: hsl(142 76% 30%);
  border: 1px solid hsl(142 76% 36% / 0.2);
}

.message-error {
  background: hsl(var(--destructive) / 0.08);
  color: hsl(var(--destructive));
  border: 1px solid hsl(var(--destructive) / 0.2);
}

/* Action Buttons */
.action-buttons {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  padding: 1.25rem 0 0 0;
  background: transparent;
  border-top: 1px solid hsl(var(--border));
  margin-top: 1.25rem;
}

.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 6px;
  font-weight: 500;
  font-size: 0.75rem;
  cursor: pointer;
  transition: all 0.15s;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  background: hsl(var(--primary));
  color: hsl(var(--primary-foreground));
}

.btn-primary:not(:disabled):hover {
  background: hsl(var(--primary) / 0.9);
}

.btn-outline {
  background: transparent;
  border: 1px solid hsl(var(--border));
  color: hsl(var(--foreground));
}

.btn-outline:not(:disabled):hover {
  background: hsl(var(--muted) / 0.4);
}

.btn-danger-outline {
  background: transparent;
  border: 1px solid hsl(var(--destructive) / 0.5);
  color: hsl(var(--destructive));
}

.btn-danger-outline:not(:disabled):hover {
  background: hsl(var(--destructive) / 0.08);
  border-color: hsl(var(--destructive));
}
</style>
