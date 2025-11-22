<template>
  <div class="fixed top-4 right-4 bg-white rounded-lg shadow-sm w-[440px] lg:w-[480px] max-h-[92vh] flex flex-col pointer-events-auto z-[1000000] border border-gray-200" @click.stop>
    <!-- Header (Fixed) -->
    <div class="flex-shrink-0 px-4 py-3 border-b border-gray-200 bg-white">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <svg class="w-4 h-4 text-gray-700" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122"/>
          </svg>
          <h2 class="text-sm font-semibold text-gray-900">Element Selector</h2>
        </div>
        <Button
          v-if="showFieldForm"
          @click="closeFieldForm"
          variant="ghost"
          size="sm"
          class="h-7 w-7 p-0 hover:bg-gray-100"
          title="Close form"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
          </svg>
        </Button>
      </div>
    </div>

    <!-- Main Content Area -->
    <div class="flex-1 flex flex-col min-h-0">
      <!-- Selected Fields Section (Always Visible) -->
      <div v-if="!showFieldForm" class="flex-1 flex flex-col min-h-0">
        <!-- Selected Fields Header -->
        <div class="flex-shrink-0 px-4 py-3 border-b border-gray-200 bg-gray-50">
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-semibold text-gray-900">
              Selected Fields ({{ props.selectedFields.length }})
            </h3>
            <Button
              @click="openAddFieldForm"
              size="sm"
              class="h-7 px-3 text-xs font-medium"
            >
              <svg class="w-3.5 h-3.5 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
              </svg>
              Add Field
            </Button>
          </div>
        </div>

        <!-- Selected Fields List (Scrollable) -->
        <ScrollArea class="flex-1">
          <div class="px-4 py-3">
            <!-- Empty State -->
            <div v-if="props.selectedFields.length === 0" class="text-center py-12">
              <div class="text-gray-400 mb-3">
                <svg class="w-16 h-16 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
                </svg>
              </div>
              <div class="font-medium text-gray-700 text-sm">No fields selected yet</div>
              <div class="text-xs mt-1 text-gray-500">Click "Add Field" to get started</div>
            </div>

            <!-- Field Cards -->
            <div v-else class="space-y-2">
              <Card
                v-for="field in props.selectedFields"
                :key="field.id"
                class="cursor-pointer hover:border-gray-400 transition-colors border-l-[3px] bg-white group"
                :class="getFieldBorderClass(field)"
                @click="openEditFieldForm(field)"
              >
                <CardContent class="p-3">
                  <div class="flex items-start justify-between gap-2">
                    <div class="flex-1 min-w-0">
                      <!-- Field Name & Badges -->
                      <div class="flex items-center gap-1.5 mb-1.5 flex-wrap">
                        <div class="font-semibold text-gray-900 text-sm">{{ field.name }}</div>
                        
                        <!-- Mode Badge -->
                        <Badge
                          v-if="field.mode === 'key-value-pairs'"
                          variant="secondary"
                          class="bg-gray-100 text-gray-700 text-[10px] font-medium h-4 px-1.5"
                        >
                          K-V
                        </Badge>
                        <Badge
                          v-else-if="field.matchCount && field.matchCount > 1"
                          variant="secondary"
                          class="bg-gray-100 text-gray-700 text-[10px] font-medium h-4 px-1.5"
                        >
                          {{ field.matchCount }}
                        </Badge>
                        <Badge
                          v-else
                          variant="outline"
                          class="text-[10px] font-medium h-4 px-1.5"
                          :class="getFieldTypeBadgeClass(field)"
                        >
                          {{ field.type }}
                        </Badge>
                        
                        <!-- Transform indicator -->
                        <Badge
                          v-if="field.transforms && Object.keys(field.transforms).length > 0"
                          variant="secondary"
                          class="bg-gray-900 text-white text-[10px] font-medium h-4 px-1.5"
                        >
                          {{ Object.keys(field.transforms).length }}
                        </Badge>
                      </div>
                      
                      <!-- Selector Display -->
                      <div v-if="field.mode === 'key-value-pairs' && field.attributes?.extractions?.[0]" class="text-[11px] font-mono mt-1.5 space-y-0.5 bg-gray-50 p-1.5 rounded border border-gray-200">
                        <div class="text-gray-700 truncate">K: {{ field.attributes.extractions[0].key_selector }}</div>
                        <div class="text-gray-700 truncate">V: {{ field.attributes.extractions[0].value_selector }}</div>
                      </div>
                      <div v-else class="text-[11px] text-gray-600 font-mono truncate mt-1.5 bg-gray-50 px-2 py-1 rounded border border-gray-200">
                        {{ field.selector }}
                      </div>
                      
                      <!-- Match Count -->
                      <div v-if="field.matchCount && field.mode !== 'key-value-pairs'" class="flex items-center gap-1.5 mt-1.5">
                        <Badge variant="outline" class="text-[10px] font-medium h-4 px-1.5" :class="field.matchCount > 1 ? 'border-gray-400 text-gray-700' : 'border-gray-300 text-gray-600'">
                          {{ field.matchCount }} {{ field.matchCount === 1 ? 'match' : 'matches' }}
                        </Badge>
                      </div>
                      
                      <!-- Sample Value -->
                      <div v-if="field.sampleValue && field.mode !== 'key-value-pairs'" class="text-[11px] text-gray-600 truncate mt-1.5 bg-gray-50 px-2 py-1 rounded border border-gray-200">
                        "{{ field.sampleValue }}"
                      </div>
                    </div>
                    
                    <!-- Action Icons -->
                    <div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                      <Button
                        @click.stop="openEditFieldForm(field)"
                        variant="ghost"
                        size="sm"
                        class="h-7 w-7 p-0 hover:bg-gray-100"
                        title="Edit field"
                      >
                        <svg class="w-3.5 h-3.5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                        </svg>
                      </Button>
                      <Button
                        @click.stop="deleteConfirmField = field"
                        variant="ghost"
                        size="sm"
                        class="h-7 w-7 p-0 text-gray-400 hover:text-red-600 hover:bg-red-50"
                        title="Delete field"
                      >
                        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                        </svg>
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>
        </ScrollArea>
      </div>

      <!-- Add/Edit Field Form (Slide-out Panel) -->
      <div v-else class="flex-1 flex flex-col min-h-0">
        <!-- Form Header -->
        <div class="flex-shrink-0 px-4 py-3 border-b border-gray-200 bg-gray-50">
          <div class="flex items-center gap-2">
            <Button
              @click="closeFieldForm"
              variant="ghost"
              size="sm"
              class="h-7 w-7 p-0 hover:bg-gray-100 -ml-1"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
              </svg>
            </Button>
            <h3 class="text-sm font-semibold text-gray-900">
              {{ editingFieldId ? 'Edit Field' : 'Add Field' }}
            </h3>
          </div>
          <div v-if="editingFieldId" class="mt-2 px-2 py-1 bg-gray-100 border border-gray-300 rounded text-[11px] text-gray-700">
            Click elements on the page to update selector
          </div>
          <div v-else class="mt-2 text-xs text-gray-500">
            Click elements on the page to select
          </div>
        </div>

        <!-- Form Content (Scrollable) -->
        <ScrollArea class="flex-1">
          <div class="px-4 py-3">
            <!-- Tab Navigation -->
            <Tabs v-model="activeTab" class="w-full">
          <TabsList class="grid w-full grid-cols-2 bg-gray-50 p-0.5">
            <TabsTrigger value="regular" class="text-xs data-[state=active]:bg-white data-[state=active]:text-gray-900 data-[state=active]:shadow-sm">
              Single/Multiple
            </TabsTrigger>
            <TabsTrigger value="key-value" class="text-xs data-[state=active]:bg-white data-[state=active]:text-gray-900 data-[state=active]:shadow-sm">
              Key-Value
            </TabsTrigger>
          </TabsList>

          <!-- Tab Content - Regular Mode -->
          <TabsContent value="regular" class="space-y-3 mt-3">
            <div>
              <Label for="field-name" class="text-xs font-medium text-gray-700 mb-1.5 block">
                Field Name
              </Label>
              <Input
                id="field-name"
                :model-value="props.fieldName"
                @update:model-value="emit('update:fieldName', $event)"
                @keydown.enter="canAddField && emit('addField')"
                type="text"
                placeholder="e.g., title, price, description"
                class="h-9 text-sm"
                autofocus
              />
            </div>

            <!-- Multiple Value Option -->
            <Card class="bg-gray-50 border border-gray-200">
              <CardContent class="p-3">
                <label class="flex items-start gap-2.5 cursor-pointer group">
                  <input
                    type="checkbox"
                    v-model="extractMultiple"
                    class="mt-0.5 w-4 h-4 text-gray-900 rounded border-gray-300 focus:ring-2 focus:ring-gray-400 cursor-pointer"
                  />
                  <div class="flex-1">
                    <span class="text-sm font-medium text-gray-900">Extract Multiple Values</span>
                    <p class="text-xs text-gray-500 mt-0.5 leading-relaxed">Extract an array of values from all matching elements</p>
                  </div>
                </label>
              </CardContent>
            </Card>

            <div>
              <Label for="extract-type" class="text-xs font-medium text-gray-700 mb-1.5 block">
                Extract Type
              </Label>
              <Select
                :model-value="props.fieldType"
                @update:model-value="emit('update:fieldType', $event)"
              >
                <SelectTrigger id="extract-type" class="h-9">
                  <SelectValue placeholder="Select extraction type" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="text">Text Content</SelectItem>
                  <SelectItem value="attribute">Attribute</SelectItem>
                  <SelectItem value="html">HTML</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div v-if="props.fieldType === 'attribute'" class="animate-in slide-in-from-top-2 duration-200">
              <Label for="attribute-name" class="text-xs font-medium text-gray-700 mb-1.5 block">
                Attribute Name
              </Label>
              <Input
                id="attribute-name"
                :model-value="props.fieldAttribute"
                @update:model-value="emit('update:fieldAttribute', $event)"
                type="text"
                placeholder="e.g., href, src, data-id"
                class="h-9 text-sm font-mono"
              />
            </div>

            <!-- Transforms Section -->
            <Card class="bg-white border border-gray-200">
              <CardContent class="p-3">
                <button
                  @click="showTransforms = !showTransforms"
                  class="flex items-center justify-between w-full text-sm font-medium text-gray-900 hover:text-gray-700"
                >
                  <div class="flex items-center gap-2">
                    <span class="text-xs">Transforms</span>
                    <Badge v-if="activeTransforms.length > 0" variant="secondary" class="bg-gray-900 text-white text-[10px] h-4 px-1.5">
                      {{ activeTransforms.length }}
                    </Badge>
                  </div>
                  <svg class="w-3.5 h-3.5 transition-transform" :class="{ 'rotate-90': showTransforms }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
                  </svg>
                </button>

                <div v-if="showTransforms" class="mt-3 space-y-3 animate-in slide-in-from-top-2 duration-200">
                  <!-- Text Transforms -->
                  <div>
                    <div class="text-[11px] font-semibold text-gray-700 mb-1.5">Text Operations</div>
                    <div class="space-y-1.5">
                      <label class="flex items-center gap-2 text-xs cursor-pointer hover:bg-gray-50 p-1.5 rounded">
                        <input type="checkbox" v-model="transforms.trim" class="w-3.5 h-3.5 text-gray-900 rounded border-gray-300 focus:ring-2 focus:ring-gray-400">
                        <span>Trim whitespace</span>
                      </label>
                      <label class="flex items-center gap-2 text-xs cursor-pointer hover:bg-gray-50 p-1.5 rounded">
                        <input type="checkbox" v-model="transforms.lowercase" class="w-3.5 h-3.5 text-gray-900 rounded border-gray-300 focus:ring-2 focus:ring-gray-400">
                        <span>Lowercase</span>
                      </label>
                      <label class="flex items-center gap-2 text-xs cursor-pointer hover:bg-gray-50 p-1.5 rounded">
                        <input type="checkbox" v-model="transforms.uppercase" class="w-3.5 h-3.5 text-gray-900 rounded border-gray-300 focus:ring-2 focus:ring-gray-400">
                        <span>Uppercase</span>
                      </label>
                      <label class="flex items-center gap-2 text-xs cursor-pointer hover:bg-gray-50 p-1.5 rounded">
                        <input type="checkbox" v-model="transforms.capitalize" class="w-3.5 h-3.5 text-gray-900 rounded border-gray-300 focus:ring-2 focus:ring-gray-400">
                        <span>Capitalize first letter</span>
                      </label>
                    </div>
                  </div>

                  <!-- String Transforms -->
                  <div>
                    <div class="text-[11px] font-semibold text-gray-700 mb-1.5">String Operations</div>
                    <div class="space-y-1.5">
                      <label class="flex items-center gap-2 text-xs cursor-pointer hover:bg-gray-50 p-1.5 rounded">
                        <input type="checkbox" v-model="transforms.removeSpaces" class="w-3.5 h-3.5 text-gray-900 rounded border-gray-300 focus:ring-2 focus:ring-gray-400">
                        <span>Remove all spaces</span>
                      </label>
                      <label class="flex items-center gap-2 text-xs cursor-pointer hover:bg-gray-50 p-1.5 rounded">
                        <input type="checkbox" v-model="transforms.removeSpecialChars" class="w-3.5 h-3.5 text-gray-900 rounded border-gray-300 focus:ring-2 focus:ring-gray-400">
                        <span>Remove special characters</span>
                      </label>
                      <label class="flex items-center gap-2 text-xs cursor-pointer hover:bg-gray-50 p-1.5 rounded">
                        <input type="checkbox" v-model="transforms.removeNumbers" class="w-3.5 h-3.5 text-gray-900 rounded border-gray-300 focus:ring-2 focus:ring-gray-400">
                        <span>Remove numbers</span>
                      </label>
                      <label class="flex items-center gap-2 text-xs cursor-pointer hover:bg-gray-50 p-1.5 rounded">
                        <input type="checkbox" v-model="transforms.extractNumbers" class="w-3.5 h-3.5 text-gray-900 rounded border-gray-300 focus:ring-2 focus:ring-gray-400">
                        <span>Extract numbers only</span>
                      </label>
                    </div>
                  </div>

                  <!-- Type Transforms -->
                  <div>
                    <div class="text-[11px] font-semibold text-gray-700 mb-1.5">Type Conversion</div>
                    <div class="space-y-1.5">
                      <label class="flex items-center gap-2 text-xs cursor-pointer hover:bg-gray-50 p-1.5 rounded">
                        <input type="checkbox" v-model="transforms.toNumber" class="w-3.5 h-3.5 text-gray-900 rounded border-gray-300 focus:ring-2 focus:ring-gray-400">
                        <span>Convert to number</span>
                      </label>
                      <label class="flex items-center gap-2 text-xs cursor-pointer hover:bg-gray-50 p-1.5 rounded">
                        <input type="checkbox" v-model="transforms.toBoolean" class="w-3.5 h-3.5 text-gray-900 rounded border-gray-300 focus:ring-2 focus:ring-gray-400">
                        <span>Convert to boolean</span>
                      </label>
                    </div>
                  </div>

                  <!-- Transform Preview -->
                  <div v-if="transformedPreviewSamples.length > 0" class="border-t border-gray-200 pt-2.5 mt-2.5">
                    <div class="text-[11px] font-semibold text-gray-700 mb-1.5">Preview with Transforms</div>
                    <div class="space-y-1.5">
                      <div v-for="(sample, idx) in transformedPreviewSamples" :key="idx" class="bg-gray-50 rounded p-2 border border-gray-200 text-[11px]">
                        <div class="text-gray-500 mb-0.5">Before: <span class="font-mono text-gray-700">{{ props.livePreviewSamples[idx] }}</span></div>
                        <div class="text-gray-900 font-medium">After: <span class="font-mono">{{ sample }}</span></div>
                      </div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>

            <!-- Validation Message -->
            <div v-if="props.hoveredElementValidation" class="text-sm animate-in fade-in duration-200">
              <Alert
                :variant="props.hoveredElementValidation.isValid ? 'default' : 'destructive'"
                class="py-2.5 border"
              >
                <AlertDescription class="text-xs font-medium">
                  {{ props.hoveredElementValidation.message }}
                </AlertDescription>
              </Alert>
            </div>

            <!-- Selector Quality & Alternatives -->
            <div v-if="props.selectorAnalysis && props.hoveredElementCount > 0" 
                 class="animate-in slide-in-from-top-2 duration-200">
              <Card class="bg-white border border-gray-200">
                <CardContent class="p-3">
                  <div class="flex items-center gap-2 mb-2.5">
                    <h3 class="text-xs font-semibold text-gray-900">Selector Quality</h3>
                    <span class="ml-auto px-2 py-0.5 text-[10px] font-semibold rounded"
                          :class="{
                            'bg-green-100 text-green-800': props.selectorAnalysis.current.rating === 'excellent',
                            'bg-blue-100 text-blue-800': props.selectorAnalysis.current.rating === 'good',
                            'bg-yellow-100 text-yellow-800': props.selectorAnalysis.current.rating === 'fair',
                            'bg-orange-100 text-orange-800': props.selectorAnalysis.current.rating === 'poor',
                            'bg-red-100 text-red-800': props.selectorAnalysis.current.rating === 'fragile'
                          }">
                      {{ props.selectorAnalysis.current.rating }}
                    </span>
                  </div>

                  <!-- Current Selector Info -->
                  <div class="text-[11px] space-y-1 mb-2.5">
                    <div v-if="props.selectorAnalysis.current.reasons.length > 0" class="flex flex-wrap gap-1">
                      <span 
                        v-for="(reason, idx) in props.selectorAnalysis.current.reasons" 
                        :key="idx"
                        class="px-1.5 py-0.5 bg-green-50 text-green-700 rounded border border-green-200"
                      >
                        {{ reason }}
                      </span>
                    </div>
                    <div v-if="props.selectorAnalysis.current.issues.length > 0" class="flex flex-wrap gap-1">
                      <span 
                        v-for="(issue, idx) in props.selectorAnalysis.current.issues" 
                        :key="idx"
                        class="px-1.5 py-0.5 bg-red-50 text-red-700 rounded border border-red-200"
                      >
                        {{ issue }}
                      </span>
                    </div>
                  </div>

                  <!-- Alternative Selectors -->
                  <div v-if="props.selectorAnalysis.alternatives.length > 0" class="border-t border-gray-200 pt-2.5 mt-2.5">
                    <div class="text-[11px] font-semibold text-gray-700 mb-2">Better Alternatives</div>
                    <div class="space-y-1.5">
                      <button
                        v-for="(alt, idx) in props.selectorAnalysis.alternatives"
                        :key="idx"
                        @click="emit('useAlternativeSelector', alt.selector)"
                        class="w-full text-left p-2 bg-gray-50 rounded border border-gray-200 hover:border-gray-400 hover:bg-gray-100 transition-all text-[11px] group"
                      >
                        <div class="flex items-center justify-between mb-1">
                          <div class="flex items-center gap-1 flex-1 min-w-0">
                            <span class="font-mono text-gray-900 truncate">{{ alt.selector }}</span>
                          </div>
                          <div class="flex items-center gap-1 ml-2">
                            <span class="text-[10px] px-1.5 py-0.5 rounded font-semibold"
                                  :class="{
                                    'bg-green-100 text-green-800': alt.quality.rating === 'excellent',
                                    'bg-blue-100 text-blue-800': alt.quality.rating === 'good',
                                    'bg-yellow-100 text-yellow-800': alt.quality.rating === 'fair'
                                  }">
                              {{ alt.quality.rating }}
                            </span>
                          </div>
                        </div>
                        <div class="text-gray-600">{{ alt.description }}</div>
                      </button>
                    </div>
                  </div>
                  <div v-else class="border-t border-gray-200 pt-2.5 mt-2.5 text-[11px] text-gray-500 text-center">
                    No better alternatives found
                  </div>
                </CardContent>
              </Card>
            </div>

            <!-- Live Preview Section -->
            <div v-if="props.livePreviewSamples.length > 0 && props.hoveredElementCount > 0" 
                 class="animate-in slide-in-from-top-2 duration-200">
              <Card class="bg-white border border-gray-200">
                <CardContent class="p-3">
                  <div class="flex items-center gap-2 mb-2.5">
                    <h3 class="text-xs font-semibold text-gray-900">Live Preview</h3>
                    <span class="ml-auto px-2 py-0.5 bg-gray-100 text-gray-700 text-[10px] font-semibold rounded">
                      {{ props.hoveredElementCount }} {{ props.hoveredElementCount === 1 ? 'match' : 'matches' }}
                    </span>
                  </div>
                  <div class="space-y-1.5">
                    <div 
                      v-for="(sample, index) in props.livePreviewSamples" 
                      :key="index"
                      class="text-[11px] bg-gray-50 rounded p-2 border border-gray-200 font-mono text-gray-700 truncate"
                      :title="sample"
                    >
                      <span class="text-gray-500 font-semibold">{{ index + 1 }}.</span> {{ sample || '(empty)' }}
                    </div>
                    <div v-if="props.hoveredElementCount > props.livePreviewSamples.length" 
                         class="text-[11px] text-gray-500 text-center">
                      ... and {{ props.hoveredElementCount - props.livePreviewSamples.length }} more
                    </div>
                  </div>
                  <div class="mt-2.5 text-[11px] text-gray-600 flex items-center gap-1.5">
                    <span class="font-medium">Output:</span>
                    <span v-if="extractMultiple" class="font-mono bg-gray-100 px-1.5 py-0.5 rounded text-gray-700">
                      Array[{{ props.hoveredElementCount }}]
                    </span>
                    <span v-else class="font-mono bg-gray-100 px-1.5 py-0.5 rounded text-gray-700">
                      Single value
                    </span>
                  </div>
                </CardContent>
              </Card>
            </div>

            <!-- Action Buttons -->
            <div class="flex gap-2">
              <Button
                v-if="editingFieldId"
                @click="cancelEdit"
                variant="outline"
                class="flex-1 h-9 text-sm font-medium"
              >
                Cancel
              </Button>
              <Button
                @click="handleAddField"
                :disabled="!canAddField"
                class="h-9 text-sm font-medium"
                :class="editingFieldId ? 'flex-1' : 'w-full'"
              >
                <span v-if="canAddField">
                  {{ editingFieldId ? 'Update Field' : 'Add Field' }}
                </span>
                <span v-else class="text-gray-400">Select an element</span>
              </Button>
            </div>
          </TabsContent>

          <!-- Tab Content - Key-Value Pair Selector -->
          <TabsContent value="key-value" class="mt-4">
            <KeyValuePairSelector
              ref="kvSelectorRef"
              v-model:field-name="kvFieldName"
              :editing-field-id="editingFieldId"
              @add="handleAddKeyValueField"
            />
          </TabsContent>
        </Tabs>


        <!-- Detailed View Content (inside panel) -->
        <DetailedFieldContent
          v-if="props.detailedViewField"
          :field="props.detailedViewField"
          :tab="props.detailedViewTab"
          :edit-mode="props.editMode"
          :test-results="props.testResults"
          @switch-tab="emit('switchTab', $event)"
          @enable-edit="emit('enableEditMode')"
          @save-edit="emit('saveEdit', $event)"
          @cancel-edit="emit('cancelEdit')"
          @test-selector="emit('testSelector', $event)"
          @scroll-to-result="emit('scrollToResult', $event)"
        />
          </div>
        </ScrollArea>
      </div>
    </div>

    <!-- Delete Confirmation Dialog -->
    <Dialog :open="deleteConfirmField !== null" @update:open="(open) => !open && (deleteConfirmField = null)">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle class="text-base font-semibold text-gray-900">
            Delete Field?
          </DialogTitle>
          <DialogDescription class="text-sm pt-1">
            Are you sure you want to delete this field? This action cannot be undone.
          </DialogDescription>
        </DialogHeader>
        
        <div v-if="deleteConfirmField" class="my-3 p-3 bg-gray-50 rounded border border-gray-200">
          <div class="font-semibold text-gray-900 text-sm mb-2">
            {{ deleteConfirmField.name }}
          </div>
          <div class="text-xs text-gray-600 space-y-1.5">
            <div class="flex items-center gap-2">
              <span class="font-medium">Type:</span>
              <Badge variant="outline" :class="getFieldTypeBadgeClass(deleteConfirmField)" class="text-[10px] h-4 px-1.5">
                {{ deleteConfirmField.type }}
              </Badge>
            </div>
            <div v-if="deleteConfirmField.mode === 'key-value-pairs'" class="flex items-center gap-2">
              <span class="font-medium">Mode:</span>
              <Badge variant="secondary" class="bg-gray-100 text-gray-700 text-[10px] h-4 px-1.5">
                Key-Value Pairs
              </Badge>
            </div>
            <div v-if="deleteConfirmField.matchCount" class="flex items-center gap-2">
              <span class="font-medium">Matches:</span>
              <span>{{ deleteConfirmField.matchCount }}</span>
            </div>
            <div v-if="deleteConfirmField.transforms && Object.keys(deleteConfirmField.transforms).length > 0" class="flex items-center gap-2">
              <span class="font-medium">Transforms:</span>
              <Badge variant="secondary" class="bg-gray-900 text-white text-[10px] h-4 px-1.5">
                {{ Object.keys(deleteConfirmField.transforms).length }}
              </Badge>
            </div>
            <div class="font-mono text-[11px] bg-white p-2 rounded border border-gray-200 mt-2 truncate">
              {{ deleteConfirmField.selector }}
            </div>
          </div>
        </div>

        <DialogFooter class="flex gap-2 sm:gap-2">
          <Button
            @click="deleteConfirmField = null"
            variant="outline"
            class="flex-1 h-9 text-sm"
          >
            Cancel
          </Button>
          <Button
            @click="confirmDelete"
            variant="destructive"
            class="flex-1 h-9 text-sm"
          >
            Delete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { SelectedField, FieldType, ValidationResult, TestResult, SelectionMode } from '../types'
import type { AlternativeSelector, SelectorQuality } from '../utils/selectorGenerator'
import DetailedFieldContent from './DetailedFieldContent.vue'
import KeyValuePairSelector from './KeyValuePairSelector.vue'
import { getElementColor } from '../utils/elementColors'

// Shadcn Components
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Label } from './ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select'
import { Card, CardContent } from './ui/card'
import { Badge } from './ui/badge'
import { Alert, AlertDescription } from './ui/alert'
import { ScrollArea } from './ui/scroll-area'
import { Tabs, TabsContent, TabsList, TabsTrigger } from './ui/tabs'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from './ui/dialog'

interface Props {
  fieldName: string
  fieldType: FieldType
  fieldAttribute: string
  mode: SelectionMode
  selectedFields: SelectedField[]
  hoveredElementCount: number
  hoveredElementValidation: ValidationResult | null
  livePreviewSamples: string[]
  selectorAnalysis: {
    current: SelectorQuality & { matchCount: number }
    alternatives: AlternativeSelector[]
  } | null
  detailedViewField: SelectedField | null
  detailedViewTab: 'preview' | 'edit'
  editMode: boolean
  testResults: TestResult[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:fieldName': [name: string]
  'update:fieldType': [type: FieldType]
  'update:fieldAttribute': [attr: string]
  'update:mode': [mode: SelectionMode]
  'addField': [transforms: any]
  'updateField': [data: { id: string; transforms: any }]
  'updateKVField': [data: { id: string; fieldName: string; extractions: any[] }]
  'loadFieldForEdit': [field: SelectedField]
  'loadKVFieldForEdit': [field: SelectedField]
  'addKeyValueField': [data: any]
  'removeField': [id: string]
  'openDetailedView': [field: SelectedField]
  'closeDetailedView': []
  'switchTab': [tab: 'preview' | 'edit']
  'enableEditMode': []
  'saveEdit': [field: Partial<SelectedField>]
  'cancelEdit': []
  'testSelector': [field: SelectedField]
  'scrollToResult': [result: TestResult]
  'useAlternativeSelector': [selector: string]
  'dialogStateChange': [open: boolean]
}>()

const activeTab = ref<'regular' | 'key-value'>('regular')
const extractMultiple = ref(false)
const kvFieldName = ref('')
const kvSelectorRef = ref<InstanceType<typeof KeyValuePairSelector> | null>(null)
const showLegend = ref(false)
const showTransforms = ref(false)
const editingFieldId = ref<string | null>(null)
const deleteConfirmField = ref<SelectedField | null>(null)
const showFieldForm = ref(false)

// Transform options
const transforms = ref({
  // Text transforms
  trim: false,
  lowercase: false,
  uppercase: false,
  capitalize: false,
  // String transforms
  removeSpaces: false,
  removeSpecialChars: false,
  removeNumbers: false,
  extractNumbers: false,
  // Type transforms
  toNumber: false,
  toBoolean: false
})

// Update mode based on active tab
watch(activeTab, (tab) => {
  const mode = tab === 'key-value' ? 'key-value-pairs' : extractMultiple.value ? 'list' : 'single'
  emit('update:mode', mode)
})

// Update mode when extractMultiple changes
watch(extractMultiple, (isMultiple) => {
  if (activeTab.value === 'regular') {
    const mode = isMultiple ? 'list' : 'single'
    emit('update:mode', mode)
  }
})

// Notify parent when dialog state changes
watch(deleteConfirmField, (field) => {
  emit('dialogStateChange', field !== null)
})

const canAddField = computed(() => {
  if (!props.fieldName.trim()) return false
  if (props.hoveredElementCount === 0) return false
  if (props.fieldType === 'attribute' && !props.fieldAttribute.trim()) return false
  return true
})

// Get list of active transforms
const activeTransforms = computed(() => {
  return Object.entries(transforms.value)
    .filter(([_, enabled]) => enabled)
    .map(([name]) => name)
})

// Apply transforms to a value
const applyTransforms = (value: string): string => {
  let result = value

  // Text transforms
  if (transforms.value.trim) {
    result = result.trim()
  }
  if (transforms.value.lowercase) {
    result = result.toLowerCase()
  }
  if (transforms.value.uppercase) {
    result = result.toUpperCase()
  }
  if (transforms.value.capitalize) {
    result = result.charAt(0).toUpperCase() + result.slice(1).toLowerCase()
  }

  // String transforms
  if (transforms.value.removeSpaces) {
    result = result.replace(/\s+/g, '')
  }
  if (transforms.value.removeSpecialChars) {
    result = result.replace(/[^a-zA-Z0-9\s]/g, '')
  }
  if (transforms.value.removeNumbers) {
    result = result.replace(/\d+/g, '')
  }
  if (transforms.value.extractNumbers) {
    const numbers = result.match(/\d+/g)
    result = numbers ? numbers.join('') : ''
  }

  // Type transforms
  if (transforms.value.toNumber) {
    const num = parseFloat(result.replace(/[^0-9.-]/g, ''))
    result = isNaN(num) ? '0' : num.toString()
  }
  if (transforms.value.toBoolean) {
    const truthyValues = ['true', 'yes', '1', 'on']
    result = truthyValues.includes(result.toLowerCase()) ? 'true' : 'false'
  }

  return result
}

// Get transformed preview samples
const transformedPreviewSamples = computed(() => {
  if (!props.livePreviewSamples || props.livePreviewSamples.length === 0) {
    return []
  }
  if (activeTransforms.value.length === 0) {
    return []
  }
  return props.livePreviewSamples.map(sample => applyTransforms(sample))
})

function handleAddField() {
  // Collect enabled transforms
  const enabledTransforms = Object.entries(transforms.value)
    .filter(([_, enabled]) => enabled)
    .reduce((acc, [key, _]) => {
      acc[key] = true
      return acc
    }, {} as Record<string, boolean>)
  
  if (editingFieldId.value) {
    // Update existing field
    emit('updateField', {
      id: editingFieldId.value,
      transforms: enabledTransforms
    })
  } else {
    // Add new field
    emit('addField', enabledTransforms)
  }
  
  // Close form and reset
  closeFieldForm()
}

function openAddFieldForm() {
  showFieldForm.value = true
  editingFieldId.value = null
  activeTab.value = 'regular'
  
  // Reset form
  emit('update:fieldName', '')
  emit('update:fieldAttribute', '')
  extractMultiple.value = false
  
  // Reset transforms
  Object.keys(transforms.value).forEach(key => {
    transforms.value[key as keyof typeof transforms.value] = false
  })
  showTransforms.value = false
}

function openEditFieldForm(field: SelectedField) {
  showFieldForm.value = true
  editingFieldId.value = field.id
  
  if (field.mode === 'key-value-pairs') {
    // Switch to Key-Value tab
    activeTab.value = 'key-value'
    
    // Populate K-V field name
    kvFieldName.value = field.name
    
    // Load the K-V field data into the selector
    emit('loadKVFieldForEdit', field)
    return
  }
  
  // Regular field editing
  // Switch to regular tab
  activeTab.value = 'regular'
  
  // Populate form with field data
  emit('update:fieldName', field.name)
  emit('update:fieldType', field.type)
  if (field.attribute) {
    emit('update:fieldAttribute', field.attribute)
  }
  
  // Set extract multiple based on mode
  extractMultiple.value = field.mode === 'list'
  
  // Populate transforms
  if (field.transforms) {
    Object.keys(transforms.value).forEach(key => {
      transforms.value[key as keyof typeof transforms.value] = field.transforms?.[key] || false
    })
    // Open transforms section if field has transforms
    if (Object.keys(field.transforms).length > 0) {
      showTransforms.value = true
    }
  }
  
  // Load the element selector for this field
  emit('loadFieldForEdit', field)
}

function closeFieldForm() {
  showFieldForm.value = false
  cancelEdit()
}

function startEditField(field: SelectedField) {
  // This function is kept for backwards compatibility but now calls openEditFieldForm
  openEditFieldForm(field)
}

function cancelEdit() {
  editingFieldId.value = null
  
  // Reset form
  emit('update:fieldName', '')
  emit('update:fieldAttribute', '')
  extractMultiple.value = false
  
  // Reset transforms
  Object.keys(transforms.value).forEach(key => {
    transforms.value[key as keyof typeof transforms.value] = false
  })
  showTransforms.value = false
  
  // Clear locked element
  emit('cancelEdit')
}

function confirmDelete() {
  if (deleteConfirmField.value) {
    emit('removeField', deleteConfirmField.value.id)
    deleteConfirmField.value = null
  }
}

function handleAddKeyValueField(data: { fieldName: string; extractions: any[] }) {
  if (editingFieldId.value) {
    // Update existing K-V field
    emit('updateKVField', {
      id: editingFieldId.value,
      fieldName: data.fieldName,
      extractions: data.extractions
    })
  } else {
    // Add new K-V field
    emit('addKeyValueField', data)
  }
  
  // Close form and reset
  kvFieldName.value = ''
  closeFieldForm()
}

const getFieldBorderClass = (field: SelectedField) => {
  if (field.type === 'text') return 'border-l-blue-500'
  if (field.type === 'attribute') return 'border-l-purple-500'
  if (field.type === 'html') return 'border-l-pink-500'
  return 'border-l-gray-500'
}

const getFieldTypeBadgeClass = (field: SelectedField) => {
  if (field.type === 'text') return 'border-blue-300 text-blue-700'
  if (field.type === 'attribute') return 'border-purple-300 text-purple-700'
  if (field.type === 'html') return 'border-pink-300 text-pink-700'
  return 'border-gray-300 text-gray-700'
}

// Expose kvSelectorRef to parent
defineExpose({
  kvSelectorRef
})
</script>
