<template>
  <div class="fixed top-4 right-4 bg-white rounded-2xl shadow-2xl w-[440px] lg:w-[480px] max-h-[92vh] flex flex-col pointer-events-auto z-[1000000] border-2 border-gray-100 overflow-hidden backdrop-blur-sm" @click.stop>
    <!-- Header (Fixed) with better gradient -->
    <div class="flex-shrink-0 px-6 pt-5 pb-4 border-b-2 border-gray-100 bg-gradient-to-br from-blue-50 via-indigo-50 to-purple-50">
      <div class="flex items-start justify-between">
        <div class="flex-1">
          <div class="flex items-center gap-3">
            <div class="flex items-center gap-2">
              <span class="text-2xl">🎯</span>
              <h2 class="text-xl font-bold text-gray-900 tracking-tight">Element Selector</h2>
            </div>
            <Button
              v-if="props.detailedViewField"
              @click="emit('closeDetailedView')"
              variant="ghost"
              size="sm"
              class="h-8 w-8 p-0 hover:bg-white/80 transition-all"
              title="Back to list (ESC)"
            >
              <span class="text-xl">←</span>
            </Button>
          </div>
          <p class="text-sm text-gray-600 mt-1.5 font-medium">
            <span v-if="editingFieldId">✏️ Editing field</span>
            <span v-else-if="props.detailedViewField">✨ Configure field details</span>
            <span v-else>👆 Click elements on the page to select</span>
          </p>
          <div v-if="editingFieldId" class="mt-2 px-3 py-1.5 bg-blue-100 border-2 border-blue-300 rounded-lg text-xs font-semibold text-blue-800">
            📝 Editing mode - Update field or cancel to create new
          </div>
        </div>
      </div>
      
      <!-- Keyboard hints - More visible and attractive -->
      <div v-if="!props.detailedViewField" class="mt-3 flex items-center gap-4 text-xs">
        <div class="flex items-center gap-1.5 text-gray-700">
          <kbd class="px-2 py-1 bg-white border-2 border-gray-300 rounded-md text-gray-800 font-mono font-bold shadow-sm">ESC</kbd>
          <span class="font-medium">Clear</span>
        </div>
        <div class="flex items-center gap-1.5 text-gray-700">
          <kbd class="px-2 py-1 bg-white border-2 border-gray-300 rounded-md text-gray-800 font-mono font-bold shadow-sm">↵</kbd>
          <span class="font-medium">Add Field</span>
        </div>
      </div>
    </div>

    <!-- Scrollable Content Area with better styling -->
    <ScrollArea class="flex-1">
      <div class="px-6 pb-6">
        <!-- Tab Navigation -->
        <Tabs v-if="!props.detailedViewField" v-model="activeTab" class="w-full mt-4">
          <TabsList class="grid w-full grid-cols-2">
            <TabsTrigger value="regular" class="flex items-center gap-1.5">
              <span>📄</span>
              <span class="text-xs">Single/Multiple</span>
            </TabsTrigger>
            <TabsTrigger value="key-value" class="flex items-center gap-1.5">
              <span>🔗</span>
              <span class="text-xs">Key-Value</span>
            </TabsTrigger>
          </TabsList>

          <!-- Tab Content - Regular Mode with improved design -->
          <TabsContent value="regular" class="space-y-4 mt-4">
            <div>
              <Label for="field-name" class="text-sm font-semibold mb-2 flex items-center gap-2">
                <span>📝</span>
                <span>Field Name</span>
              </Label>
              <Input
                id="field-name"
                :model-value="props.fieldName"
                @update:model-value="emit('update:fieldName', $event)"
                @keydown.enter="canAddField && emit('addField')"
                type="text"
                placeholder="e.g., title, price, description"
                class="mt-1.5 h-11 text-base"
                autofocus
              />
            </div>

            <!-- Multiple Value Option with better styling -->
            <Card class="bg-gradient-to-br from-blue-50 to-indigo-50 border-2 border-blue-200 shadow-sm hover:shadow-md transition-shadow">
              <CardContent class="p-4">
                <label class="flex items-start gap-3 cursor-pointer group">
                  <input
                    type="checkbox"
                    v-model="extractMultiple"
                    class="mt-0.5 w-5 h-5 text-blue-600 rounded-md border-gray-300 focus:ring-2 focus:ring-blue-500 cursor-pointer"
                  />
                  <div class="flex-1">
                    <span class="text-sm font-semibold text-gray-900 group-hover:text-blue-700 transition-colors">📋 Extract Multiple Values</span>
                    <p class="text-xs text-gray-600 mt-1 leading-relaxed">Extract an array of values from all matching elements on the page</p>
                  </div>
                </label>
              </CardContent>
            </Card>

            <div>
              <Label for="extract-type" class="text-sm font-semibold mb-2 flex items-center gap-2">
                <span>🎨</span>
                <span>Extract Type</span>
              </Label>
              <Select
                :model-value="props.fieldType"
                @update:model-value="emit('update:fieldType', $event)"
              >
                <SelectTrigger id="extract-type" class="mt-1.5 h-11">
                  <SelectValue placeholder="Select extraction type" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="text">📝 Text Content</SelectItem>
                  <SelectItem value="attribute">🏷️ Attribute</SelectItem>
                  <SelectItem value="html">📄 HTML</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div v-if="props.fieldType === 'attribute'" class="animate-in slide-in-from-top-2 duration-300">
              <Label for="attribute-name" class="text-sm font-semibold mb-2 flex items-center gap-2">
                <span>🏷️</span>
                <span>Attribute Name</span>
              </Label>
              <Input
                id="attribute-name"
                :model-value="props.fieldAttribute"
                @update:model-value="emit('update:fieldAttribute', $event)"
                type="text"
                placeholder="e.g., href, src, data-id"
                class="mt-1.5 h-11 text-base font-mono"
              />
            </div>

            <!-- Transforms Section -->
            <Card class="bg-gradient-to-br from-amber-50 to-orange-50 border-2 border-amber-200 shadow-sm">
              <CardContent class="p-4">
                <button
                  @click="showTransforms = !showTransforms"
                  class="flex items-center justify-between w-full text-sm font-bold text-gray-900 hover:text-amber-700 transition-colors"
                >
                  <div class="flex items-center gap-2">
                    <span>✨</span>
                    <span>Transforms</span>
                    <Badge v-if="activeTransforms.length > 0" variant="secondary" class="bg-amber-600 text-white text-xs">
                      {{ activeTransforms.length }}
                    </Badge>
                  </div>
                  <span class="text-xs">{{ showTransforms ? '▼' : '▶' }}</span>
                </button>

                <div v-if="showTransforms" class="mt-4 space-y-3 animate-in slide-in-from-top-2 duration-300">
                  <!-- Text Transforms -->
                  <div>
                    <div class="text-xs font-bold text-gray-700 mb-2">Text Operations</div>
                    <div class="space-y-2">
                      <label class="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/50 p-2 rounded transition-colors">
                        <input type="checkbox" v-model="transforms.trim" class="w-4 h-4 text-amber-600 rounded border-gray-300 focus:ring-2 focus:ring-amber-500">
                        <span>✂️ Trim whitespace</span>
                      </label>
                      <label class="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/50 p-2 rounded transition-colors">
                        <input type="checkbox" v-model="transforms.lowercase" class="w-4 h-4 text-amber-600 rounded border-gray-300 focus:ring-2 focus:ring-amber-500">
                        <span>🔡 Lowercase</span>
                      </label>
                      <label class="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/50 p-2 rounded transition-colors">
                        <input type="checkbox" v-model="transforms.uppercase" class="w-4 h-4 text-amber-600 rounded border-gray-300 focus:ring-2 focus:ring-amber-500">
                        <span>🔠 Uppercase</span>
                      </label>
                      <label class="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/50 p-2 rounded transition-colors">
                        <input type="checkbox" v-model="transforms.capitalize" class="w-4 h-4 text-amber-600 rounded border-gray-300 focus:ring-2 focus:ring-amber-500">
                        <span>Ⓐ Capitalize first letter</span>
                      </label>
                    </div>
                  </div>

                  <!-- String Transforms -->
                  <div>
                    <div class="text-xs font-bold text-gray-700 mb-2">String Operations</div>
                    <div class="space-y-2">
                      <label class="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/50 p-2 rounded transition-colors">
                        <input type="checkbox" v-model="transforms.removeSpaces" class="w-4 h-4 text-amber-600 rounded border-gray-300 focus:ring-2 focus:ring-amber-500">
                        <span>🚫 Remove all spaces</span>
                      </label>
                      <label class="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/50 p-2 rounded transition-colors">
                        <input type="checkbox" v-model="transforms.removeSpecialChars" class="w-4 h-4 text-amber-600 rounded border-gray-300 focus:ring-2 focus:ring-amber-500">
                        <span>🔧 Remove special characters</span>
                      </label>
                      <label class="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/50 p-2 rounded transition-colors">
                        <input type="checkbox" v-model="transforms.removeNumbers" class="w-4 h-4 text-amber-600 rounded border-gray-300 focus:ring-2 focus:ring-amber-500">
                        <span>🔢 Remove numbers</span>
                      </label>
                      <label class="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/50 p-2 rounded transition-colors">
                        <input type="checkbox" v-model="transforms.extractNumbers" class="w-4 h-4 text-amber-600 rounded border-gray-300 focus:ring-2 focus:ring-amber-500">
                        <span>🎯 Extract numbers only</span>
                      </label>
                    </div>
                  </div>

                  <!-- Type Transforms -->
                  <div>
                    <div class="text-xs font-bold text-gray-700 mb-2">Type Conversion</div>
                    <div class="space-y-2">
                      <label class="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/50 p-2 rounded transition-colors">
                        <input type="checkbox" v-model="transforms.toNumber" class="w-4 h-4 text-amber-600 rounded border-gray-300 focus:ring-2 focus:ring-amber-500">
                        <span>💯 Convert to number</span>
                      </label>
                      <label class="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/50 p-2 rounded transition-colors">
                        <input type="checkbox" v-model="transforms.toBoolean" class="w-4 h-4 text-amber-600 rounded border-gray-300 focus:ring-2 focus:ring-amber-500">
                        <span>✔️ Convert to boolean</span>
                      </label>
                    </div>
                  </div>

                  <!-- Transform Preview -->
                  <div v-if="transformedPreviewSamples.length > 0" class="border-t border-amber-300 pt-3 mt-3">
                    <div class="text-xs font-bold text-gray-700 mb-2">👁️ Preview with Transforms</div>
                    <div class="space-y-2">
                      <div v-for="(sample, idx) in transformedPreviewSamples" :key="idx" class="bg-white rounded p-2 border border-amber-300 text-xs">
                        <div class="text-gray-500 mb-1">Before: <span class="font-mono">{{ props.livePreviewSamples[idx] }}</span></div>
                        <div class="text-amber-700 font-semibold">After: <span class="font-mono">{{ sample }}</span></div>
                      </div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>

            <!-- Validation Message with better styling -->
            <div v-if="props.hoveredElementValidation" class="text-sm animate-in fade-in duration-200">
              <Alert
                :variant="props.hoveredElementValidation.isValid ? 'default' : 'destructive'"
                class="py-3 shadow-sm"
              >
                <span class="text-lg mr-2">{{ props.hoveredElementValidation.isValid ? '✅' : '❌' }}</span>
                <AlertDescription class="font-semibold">
                  {{ props.hoveredElementValidation.message }}
                </AlertDescription>
              </Alert>
            </div>

            <!-- Selector Quality & Alternatives -->
            <div v-if="props.selectorAnalysis && props.hoveredElementCount > 0" 
                 class="animate-in slide-in-from-top-2 duration-300">
              <Card class="bg-gradient-to-br from-purple-50 to-indigo-50 border-2 border-purple-200 shadow-sm">
                <CardContent class="p-4">
                  <div class="flex items-center gap-2 mb-3">
                    <span class="text-lg">⭐</span>
                    <h3 class="text-sm font-bold text-gray-900">SELECTOR QUALITY</h3>
                    <span class="ml-auto px-2 py-0.5 text-xs font-bold rounded-full"
                          :class="{
                            'bg-green-600 text-white': props.selectorAnalysis.current.rating === 'excellent',
                            'bg-blue-600 text-white': props.selectorAnalysis.current.rating === 'good',
                            'bg-yellow-600 text-white': props.selectorAnalysis.current.rating === 'fair',
                            'bg-orange-600 text-white': props.selectorAnalysis.current.rating === 'poor',
                            'bg-red-600 text-white': props.selectorAnalysis.current.rating === 'fragile'
                          }">
                      {{ props.selectorAnalysis.current.rating.toUpperCase() }}
                    </span>
                  </div>

                  <!-- Current Selector Info -->
                  <div class="text-xs space-y-1 mb-3">
                    <div v-if="props.selectorAnalysis.current.reasons.length > 0" class="flex flex-wrap gap-1">
                      <span 
                        v-for="(reason, idx) in props.selectorAnalysis.current.reasons" 
                        :key="idx"
                        class="px-2 py-0.5 bg-green-100 text-green-800 rounded-full font-medium"
                      >
                        ✓ {{ reason }}
                      </span>
                    </div>
                    <div v-if="props.selectorAnalysis.current.issues.length > 0" class="flex flex-wrap gap-1">
                      <span 
                        v-for="(issue, idx) in props.selectorAnalysis.current.issues" 
                        :key="idx"
                        class="px-2 py-0.5 bg-red-100 text-red-800 rounded-full font-medium"
                      >
                        ⚠ {{ issue }}
                      </span>
                    </div>
                  </div>

                  <!-- Alternative Selectors -->
                  <div v-if="props.selectorAnalysis.alternatives.length > 0" class="border-t border-purple-300 pt-3 mt-3">
                    <div class="text-xs font-bold text-gray-700 mb-2">💡 Better Alternatives:</div>
                    <div class="space-y-2">
                      <button
                        v-for="(alt, idx) in props.selectorAnalysis.alternatives"
                        :key="idx"
                        @click="emit('useAlternativeSelector', alt.selector)"
                        class="w-full text-left p-2 bg-white rounded border border-purple-300 hover:border-purple-500 hover:bg-purple-50 transition-all text-xs group"
                      >
                        <div class="flex items-center justify-between mb-1">
                          <div class="flex items-center gap-1">
                            <span class="font-mono text-purple-700 truncate">{{ alt.selector }}</span>
                          </div>
                          <div class="flex items-center gap-1">
                            <span class="text-[10px] px-1.5 py-0.5 rounded-full font-bold"
                                  :class="{
                                    'bg-green-500 text-white': alt.quality.rating === 'excellent',
                                    'bg-blue-500 text-white': alt.quality.rating === 'good',
                                    'bg-yellow-500 text-white': alt.quality.rating === 'fair'
                                  }">
                              {{ '⭐'.repeat(alt.quality.score) }}
                            </span>
                          </div>
                        </div>
                        <div class="text-gray-600 italic">{{ alt.description }}</div>
                        <div class="text-purple-600 font-semibold mt-1 opacity-0 group-hover:opacity-100 transition-opacity">
                          → Click to use this selector
                        </div>
                      </button>
                    </div>
                  </div>
                  <div v-else class="border-t border-purple-300 pt-3 mt-3 text-xs text-gray-600 italic text-center">
                    No better alternatives found
                  </div>
                </CardContent>
              </Card>
            </div>

            <!-- Live Preview Section -->
            <div v-if="props.livePreviewSamples.length > 0 && props.hoveredElementCount > 0" 
                 class="animate-in slide-in-from-top-2 duration-300">
              <Card class="bg-gradient-to-br from-green-50 to-emerald-50 border-2 border-green-200 shadow-sm">
                <CardContent class="p-4">
                  <div class="flex items-center gap-2 mb-3">
                    <span class="text-lg">👁️</span>
                    <h3 class="text-sm font-bold text-gray-900">LIVE PREVIEW</h3>
                    <span class="ml-auto px-2 py-0.5 bg-green-600 text-white text-xs font-bold rounded-full">
                      {{ props.hoveredElementCount }} {{ props.hoveredElementCount === 1 ? 'match' : 'matches' }}
                    </span>
                  </div>
                  <div class="space-y-2">
                    <div 
                      v-for="(sample, index) in props.livePreviewSamples" 
                      :key="index"
                      class="text-xs bg-white rounded-md p-2 border border-green-300 font-mono text-gray-700 truncate"
                      :title="sample"
                    >
                      <span class="text-green-600 font-bold">{{ index + 1 }}.</span> {{ sample || '(empty)' }}
                    </div>
                    <div v-if="props.hoveredElementCount > props.livePreviewSamples.length" 
                         class="text-xs text-gray-600 italic text-center">
                      ... and {{ props.hoveredElementCount - props.livePreviewSamples.length }} more
                    </div>
                  </div>
                  <div class="mt-3 text-xs text-gray-600 flex items-center gap-1">
                    <span class="font-semibold">Output:</span>
                    <span v-if="extractMultiple" class="font-mono bg-white px-2 py-0.5 rounded border border-green-300">
                      Array[{{ props.hoveredElementCount }}]
                    </span>
                    <span v-else class="font-mono bg-white px-2 py-0.5 rounded border border-green-300">
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
                class="flex-1 h-12 text-base font-semibold"
                size="lg"
              >
                <span class="flex items-center gap-2">
                  <span class="text-lg">✕</span>
                  <span>Cancel</span>
                </span>
              </Button>
              <Button
                @click="handleAddField"
                :disabled="!canAddField"
                class="h-12 text-base font-semibold shadow-md hover:shadow-lg transition-all"
                :class="editingFieldId ? 'flex-1' : 'w-full'"
                size="lg"
              >
                <span v-if="canAddField" class="flex items-center gap-2">
                  <span class="text-lg">{{ editingFieldId ? '💾' : '✓' }}</span>
                  <span>{{ editingFieldId ? 'Update Field' : 'Add Field' }}</span>
                </span>
                <span v-else class="text-gray-400">Select an element to continue</span>
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

        <!-- Selected Fields List with improved design -->
        <div v-if="!props.detailedViewField" class="mt-6">
          <div class="flex items-center justify-between mb-4">
            <h3 class="text-base font-bold text-gray-900 flex items-center gap-2">
              <span>📦</span>
              <span>Selected Fields</span>
            </h3>
            <Badge variant="secondary" class="text-sm px-3 py-1 font-bold bg-blue-100 text-blue-700 border border-blue-300">
              {{ props.selectedFields.length }}
            </Badge>
          </div>
          
          <div v-if="props.selectedFields.length === 0" class="text-sm text-gray-500 text-center py-10 border-2 border-dashed border-gray-300 rounded-xl bg-gray-50">
            <div class="text-5xl mb-3 animate-bounce">📋</div>
            <div class="font-semibold text-gray-700 text-base">No fields selected yet</div>
            <div class="text-xs mt-2 text-gray-600">Click on elements in the page to start selecting</div>
          </div>

          <div v-else class="space-y-3">
            <Card
              v-for="field in props.selectedFields"
              :key="field.id"
              class="cursor-pointer hover:shadow-lg hover:scale-[1.02] transition-all duration-200 border-l-4 bg-gradient-to-r from-white to-gray-50"
              :class="[getFieldBorderClass(field), editingFieldId === field.id ? 'ring-4 ring-blue-400 ring-offset-2' : '']"
              @click="startEditField(field)"
            >
              <CardContent class="p-4">
                <div class="flex items-start justify-between gap-3">
                  <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2 mb-2">
                      <div class="font-bold text-gray-900 truncate text-base">{{ field.name }}</div>
                      <!-- Mode Badge with improved styling -->
                      <Badge
                        v-if="field.mode === 'key-value-pairs'"
                        variant="secondary"
                        class="bg-gradient-to-r from-purple-100 to-purple-200 text-purple-800 border-purple-300 text-xs font-bold"
                      >
                        🔗 K-V
                      </Badge>
                      <Badge
                        v-else-if="field.matchCount && field.matchCount > 1"
                        variant="secondary"
                        class="bg-gradient-to-r from-purple-100 to-indigo-100 text-purple-800 border-purple-300 text-xs font-bold"
                      >
                        📋 {{ field.matchCount }}
                      </Badge>
                      <Badge
                        v-else
                        variant="outline"
                        class="text-xs font-semibold"
                        :class="getFieldTypeBadgeClass(field)"
                      >
                        {{ field.type }}
                      </Badge>
                      <!-- Transform indicator -->
                      <Badge
                        v-if="field.transforms && Object.keys(field.transforms).length > 0"
                        variant="secondary"
                        class="bg-gradient-to-r from-amber-100 to-orange-100 text-amber-800 border-amber-300 text-xs font-bold"
                      >
                        ✨ {{ Object.keys(field.transforms).length }}
                      </Badge>
                    </div>
                    
                    <!-- Selector Display with better contrast -->
                    <div v-if="field.mode === 'key-value-pairs' && field.attributes?.extractions?.[0]" class="text-xs font-mono mt-2 space-y-1 bg-gray-100 p-2 rounded border border-gray-200">
                      <div class="text-green-700 truncate font-semibold">🔑 {{ field.attributes.extractions[0].key_selector }}</div>
                      <div class="text-blue-700 truncate font-semibold">💎 {{ field.attributes.extractions[0].value_selector }}</div>
                    </div>
                    <div v-else class="text-xs text-gray-600 font-mono truncate mt-2 bg-gray-100 px-2 py-1.5 rounded border border-gray-200">
                      {{ field.selector }}
                    </div>
                    
                    <div v-if="field.matchCount && field.mode !== 'key-value-pairs'" class="flex items-center gap-2 mt-2">
                      <Badge variant="outline" class="text-xs font-semibold" :class="field.matchCount > 1 ? 'border-purple-400 text-purple-700 bg-purple-50' : 'border-blue-400 text-blue-700 bg-blue-50'">
                        {{ field.matchCount }} {{ field.matchCount === 1 ? 'match' : 'matches' }}
                      </Badge>
                    </div>
                    <div v-if="field.sampleValue && field.mode !== 'key-value-pairs'" class="text-xs text-gray-700 truncate mt-2 italic bg-blue-50 px-2 py-1 rounded border border-blue-200">
                      💬 "{{ field.sampleValue }}"
                    </div>
                  </div>
                  <Button
                    @click.stop="deleteConfirmField = field"
                    variant="ghost"
                    size="sm"
                    class="h-8 w-8 p-0 ml-2 text-red-500 hover:text-red-700 hover:bg-red-100 rounded-lg transition-all hover:scale-110"
                    title="Remove field"
                  >
                    <span class="text-lg">✕</span>
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>

          <!-- Color Legend (Collapsible) -->
          <div class="mt-4 border-t pt-4">
            <button
              @click="showLegend = !showLegend"
              class="flex items-center justify-between w-full text-sm font-semibold text-gray-700 hover:text-gray-900 transition-colors"
            >
              <div class="flex items-center gap-2">
                <span>🎨</span>
                <span>Color Legend</span>
              </div>
              <span class="text-xs">{{ showLegend ? '▼' : '▶' }}</span>
            </button>
            
            <div v-if="showLegend" class="mt-3 space-y-2 animate-in slide-in-from-top-2 duration-300">
              <div class="text-xs text-gray-600 mb-2 font-medium">Field Types:</div>
              <div class="flex items-center gap-2 text-xs">
                <div class="w-3 h-3 rounded-full bg-blue-500"></div>
                <span class="text-gray-700">Text Content</span>
              </div>
              <div class="flex items-center gap-2 text-xs">
                <div class="w-3 h-3 rounded-full bg-purple-500"></div>
                <span class="text-gray-700">Attribute</span>
              </div>
              <div class="flex items-center gap-2 text-xs">
                <div class="w-3 h-3 rounded-full bg-pink-500"></div>
                <span class="text-gray-700">HTML</span>
              </div>
              
              <div class="text-xs text-gray-600 mt-3 mb-2 font-medium">Match Count:</div>
              <div class="flex items-center gap-2 text-xs">
                <div class="w-3 h-3 rounded-full bg-green-500"></div>
                <span class="text-gray-700">1 match (unique)</span>
              </div>
              <div class="flex items-center gap-2 text-xs">
                <div class="w-3 h-3 rounded-full bg-blue-500"></div>
                <span class="text-gray-700">2-10 matches</span>
              </div>
              <div class="flex items-center gap-2 text-xs">
                <div class="w-3 h-3 rounded-full bg-orange-500"></div>
                <span class="text-gray-700">11+ matches</span>
              </div>
            </div>
          </div>
        </div>

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

    <!-- Delete Confirmation Dialog -->
    <Dialog :open="deleteConfirmField !== null" @update:open="(open) => !open && (deleteConfirmField = null)">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2 text-red-600">
            <span class="text-2xl">⚠️</span>
            <span>Delete Field?</span>
          </DialogTitle>
          <DialogDescription class="text-base pt-2">
            Are you sure you want to delete this field? This action cannot be undone.
          </DialogDescription>
        </DialogHeader>
        
        <div v-if="deleteConfirmField" class="my-4 p-4 bg-gray-50 rounded-lg border-2 border-gray-200">
          <div class="font-bold text-gray-900 text-lg mb-2 flex items-center gap-2">
            <span>📝</span>
            <span>{{ deleteConfirmField.name }}</span>
          </div>
          <div class="text-sm text-gray-600 space-y-1">
            <div class="flex items-center gap-2">
              <span class="font-semibold">Type:</span>
              <Badge variant="outline" :class="getFieldTypeBadgeClass(deleteConfirmField)">
                {{ deleteConfirmField.type }}
              </Badge>
            </div>
            <div v-if="deleteConfirmField.mode === 'key-value-pairs'" class="flex items-center gap-2">
              <span class="font-semibold">Mode:</span>
              <Badge variant="secondary" class="bg-purple-100 text-purple-800">
                🔗 Key-Value Pairs
              </Badge>
            </div>
            <div v-if="deleteConfirmField.matchCount" class="flex items-center gap-2">
              <span class="font-semibold">Matches:</span>
              <span>{{ deleteConfirmField.matchCount }}</span>
            </div>
            <div v-if="deleteConfirmField.transforms && Object.keys(deleteConfirmField.transforms).length > 0" class="flex items-center gap-2">
              <span class="font-semibold">Transforms:</span>
              <Badge variant="secondary" class="bg-amber-100 text-amber-800">
                ✨ {{ Object.keys(deleteConfirmField.transforms).length }}
              </Badge>
            </div>
            <div class="font-mono text-xs bg-white p-2 rounded border border-gray-300 mt-2 truncate">
              {{ deleteConfirmField.selector }}
            </div>
          </div>
        </div>

        <DialogFooter class="flex gap-2 sm:gap-2">
          <Button
            @click="deleteConfirmField = null"
            variant="outline"
            class="flex-1"
          >
            <span class="flex items-center gap-2">
              <span>✕</span>
              <span>Cancel</span>
            </span>
          </Button>
          <Button
            @click="confirmDelete"
            variant="destructive"
            class="flex-1"
          >
            <span class="flex items-center gap-2">
              <span>🗑️</span>
              <span>Delete</span>
            </span>
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
    cancelEdit()
  } else {
    // Add new field
    emit('addField', enabledTransforms)
  }
  
  // Reset transforms after adding/updating
  Object.keys(transforms.value).forEach(key => {
    transforms.value[key as keyof typeof transforms.value] = false
  })
  showTransforms.value = false
}

function startEditField(field: SelectedField) {
  console.log('🎯 startEditField called with field:', field)
  console.log('  - field.mode:', field.mode)
  console.log('  - field.name:', field.name)
  
  // Set editing mode
  editingFieldId.value = field.id
  
  if (field.mode === 'key-value-pairs') {
    console.log('✅ K-V field detected')
    // Switch to Key-Value tab
    activeTab.value = 'key-value'
    console.log('  - Switched to K-V tab')
    
    // Populate K-V field name
    kvFieldName.value = field.name
    console.log('  - Set field name:', kvFieldName.value)
    
    // Load the K-V field data into the selector
    console.log('  - Emitting loadKVFieldForEdit with field:', field)
    emit('loadKVFieldForEdit', field)
    
    // Scroll to top to show the form
    const scrollArea = document.querySelector('.scrollable-content')
    if (scrollArea) {
      scrollArea.scrollTo({ top: 0, behavior: 'smooth' })
    }
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
  
  // Scroll to top to show the form
  const scrollArea = document.querySelector('.scrollable-content')
  if (scrollArea) {
    scrollArea.scrollTo({ top: 0, behavior: 'smooth' })
  }
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
    cancelEdit()
  } else {
    // Add new K-V field
    emit('addKeyValueField', data)
  }
  kvFieldName.value = ''
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
