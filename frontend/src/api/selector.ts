import apiClient from './client';

export interface SelectedField {
  name: string;
  selector: string;
  type: 'text' | 'attribute' | 'html';
  attribute?: string;
  multiple: boolean;
  xpath?: string;
  preview?: string;
  mode?: 'single' | 'list' | 'key-value-pairs';
  attributes?: {
    extractions: Array<{
      key_selector: string;
      value_selector: string;
      key_type: string;
      value_type: string;
      key_attribute?: string;
      value_attribute?: string;
      transform?: string;
    }>;
  };
}

export interface SelectorSession {
  session_id: string;
  url: string;
  message?: string;
}

export interface SelectorSessionStatus {
  session_id: string;
  url: string;
  created_at: string;
  last_activity: string;
  active: boolean;
  fields_count: number;
}

export interface SelectorFieldsResponse {
  session_id: string;
  fields: SelectedField[];
  count: number;
}

export const selectorApi = {
  // Create a new selector session
  async createSession(url: string, existingFields?: SelectedField[]): Promise<SelectorSession> {
    const response = await apiClient.post('/selector/sessions', { 
      url,
      existing_fields: existingFields && existingFields.length > 0 ? existingFields : undefined
    });
    return response.data;
  },

  // Get session status
  async getSessionStatus(sessionId: string): Promise<SelectorSessionStatus> {
    const response = await apiClient.get(`/selector/sessions/${sessionId}`);
    return response.data;
  },

  // Get selected fields from a session
  async getSelectedFields(sessionId: string): Promise<SelectorFieldsResponse> {
    const response = await apiClient.get(`/selector/sessions/${sessionId}/fields`);
    return response.data;
  },

  // Close a selector session
  async closeSession(sessionId: string): Promise<void> {
    await apiClient.delete(`/selector/sessions/${sessionId}`);
  },

  // Poll for selected fields (helper method)
  async pollForFields(
    sessionId: string,
    intervalMs: number = 2000,
    onUpdate: (fields: SelectedField[]) => void,
    onError?: (error: any) => void
  ): Promise<() => void> {
    let active = true;
    let pollCount = 0;

    const poll = async () => {
      while (active) {
        try {
          pollCount++;
          console.log(`🔄 [Polling] Attempt ${pollCount} for session: ${sessionId}`);
          const result = await this.getSelectedFields(sessionId);
          console.log(`  - Response:`, result);
          console.log(`  - Fields count: ${result.fields.length}`);
          if (result.fields.length > 0) {
            console.log(`  - Fields detail:`, result.fields.map(f => ({
              name: f.name,
              mode: f.mode,
              selector: f.selector,
              hasExtractions: !!f.attributes?.extractions,
              extractionsCount: f.attributes?.extractions?.length || 0
            })));
          }
          onUpdate(result.fields);
        } catch (error) {
          console.error(`❌ [Polling] Error on attempt ${pollCount}:`, error);
          if (onError) {
            onError(error);
          }
          // Session might be closed, stop polling
          active = false;
        }
        await new Promise(resolve => setTimeout(resolve, intervalMs));
      }
    };

    console.log(`▶️ [Polling] Started for session: ${sessionId}, interval: ${intervalMs}ms`);
    poll();

    // Return a function to stop polling
    return () => {
      console.log(`⏹️ [Polling] Stopped for session: ${sessionId} after ${pollCount} attempts`);
      active = false;
    };
  }
};
