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
  max-width: 900px;
  margin: 0 auto;
}

/* Loading State */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
  text-align: center;
}

.spinner {
  width: 48px;
  height: 48px;
  border: 4px solid #e5e7eb;
  border-top-color: #3b82f6;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin-bottom: 1rem;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Settings Container */
.settings-container {
  background: hsl(var(--background));
  border-radius: 12px;
  overflow: hidden;
}

/* Header Section */
.settings-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 2rem;
  background: linear-gradient(135deg, hsl(var(--primary) / 0.05), hsl(var(--primary) / 0.02));
  border-bottom: 1px solid hsl(var(--border));
}

.header-content h3 {
  margin: 0 0 0.5rem 0;
  font-size: 1.5rem;
  font-weight: 700;
  color: hsl(var(--foreground));
}

.header-content p {
  margin: 0;
  color: hsl(var(--muted-foreground));
  font-size: 0.95rem;
}

.header-toggle {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.toggle-switch {
  position: relative;
  display: inline-block;
  width: 52px;
  height: 28px;
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
  transition: 0.3s;
  border-radius: 34px;
}

.slider:before {
  position: absolute;
  content: "";
  height: 20px;
  width: 20px;
  left: 4px;
  bottom: 4px;
  background-color: white;
  transition: 0.3s;
  border-radius: 50%;
}

.toggle-switch input:checked + .slider {
  background-color: hsl(var(--primary));
}

.toggle-switch input:checked + .slider:before {
  transform: translateX(24px);
}

.toggle-label {
  font-weight: 600;
  font-size: 0.95rem;
  color: hsl(var(--foreground));
}

/* Config Sections */
.config-section {
  display: flex;
  gap: 1.5rem;
  padding: 2rem;
  border-bottom: 1px solid hsl(var(--border));
}

.config-section:last-of-type {
  border-bottom: none;
}

.section-icon {
  font-size: 2rem;
  flex-shrink: 0;
}

.section-content {
  flex: 1;
}

.section-content h4 {
  margin: 0 0 0.5rem 0;
  font-size: 1.1rem;
  font-weight: 600;
  color: hsl(var(--foreground));
}

.section-description {
  margin: 0 0 1.5rem 0;
  color: hsl(var(--muted-foreground));
  font-size: 0.9rem;
}

/* Frequency Selector */
.frequency-selector {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 0.75rem;
  margin-bottom: 1.5rem;
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
  padding: 1rem;
  border: 2px solid hsl(var(--border));
  border-radius: 8px;
  transition: all 0.2s;
  background: hsl(var(--background));
}

.frequency-option.active .option-content {
  border-color: hsl(var(--primary));
  background: hsl(var(--primary) / 0.05);
}

.option-label {
  font-weight: 600;
  font-size: 0.95rem;
  color: hsl(var(--foreground));
  margin-bottom: 0.25rem;
}

.option-description {
  font-size: 0.8rem;
  color: hsl(var(--muted-foreground));
}

.cron-display {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  background: hsl(var(--muted) / 0.3);
  border-radius: 6px;
}

.cron-display code {
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 0.9rem;
  color: hsl(var(--foreground));
  font-weight: 600;
}

.cron-label {
  font-size: 0.8rem;
  color: hsl(var(--muted-foreground));
}

/* Form Grid */
.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 1.25rem;
}

.form-field.full-width {
  grid-column: 1 / -1;
}

.form-field label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
  font-size: 0.9rem;
  color: hsl(var(--foreground));
}

.required {
  color: hsl(var(--destructive));
}

.input-field {
  width: 100%;
  padding: 0.65rem 0.85rem;
  border: 1px solid hsl(var(--border));
  border-radius: 6px;
  font-size: 0.9rem;
  transition: all 0.2s;
  background: hsl(var(--background));
  color: hsl(var(--foreground));
}

.input-field:focus {
  outline: none;
  border-color: hsl(var(--primary));
  box-shadow: 0 0 0 3px hsl(var(--primary) / 0.1);
}

.field-hint {
  display: block;
  margin-top: 0.4rem;
  font-size: 0.8rem;
  color: hsl(var(--muted-foreground));
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
  gap: 0.75rem;
  cursor: pointer;
  padding: 1rem;
  border: 1px solid hsl(var(--border));
  border-radius: 6px;
  transition: all 0.2s;
}

.checkbox-field:hover {
  background: hsl(var(--muted) / 0.3);
}

.checkbox-field input {
  margin-top: 0.2rem;
}

.checkbox-label {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.checkbox-description {
  font-size: 0.8rem;
  color: hsl(var(--muted-foreground));
  font-weight: normal;
}

/* Test Notification */
.test-notification {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-top: 1.5rem;
  padding-top: 1.5rem;
  border-top: 1px solid hsl(var(--border) / 0.5);
}

.test-hint {
  font-size: 0.85rem;
  color: hsl(var(--muted-foreground));
}

/* Messages */
.message {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  margin: 1.5rem 2rem;
  border-radius: 8px;
  font-size: 0.9rem;
}

.message-icon {
  font-size: 1.2rem;
  font-weight: bold;
}

.message-success {
  background: hsl(142 76% 36% / 0.1);
  color: hsl(142 76% 36%);
  border: 1px solid hsl(142 76% 36% / 0.3);
}

.message-error {
  background: hsl(var(--destructive) / 0.1);
  color: hsl(var(--destructive));
  border: 1px solid hsl(var(--destructive) / 0.3);
}

/* Action Buttons */
.action-buttons {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 1.5rem 2rem;
  background: hsl(var(--muted) / 0.2);
}

.btn {
  padding: 0.65rem 1.5rem;
  border: none;
  border-radius: 6px;
  font-weight: 600;
  font-size: 0.9rem;
  cursor: pointer;
  transition: all 0.2s;
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
  transform: translateY(-1px);
  box-shadow: 0 4px 12px hsl(var(--primary) / 0.3);
}

.btn-outline {
  background: transparent;
  border: 1px solid hsl(var(--border));
  color: hsl(var(--foreground));
}

.btn-outline:not(:disabled):hover {
  background: hsl(var(--muted) / 0.5);
}

.btn-danger-outline {
  background: transparent;
  border: 1px solid hsl(var(--destructive));
  color: hsl(var(--destructive));
}

.btn-danger-outline:not(:disabled):hover {
  background: hsl(var(--destructive) / 0.1);
}
</style>
