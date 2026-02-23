<template>
  <div class="space-y-4">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h3 class="text-lg font-medium">
          {{ $t('bots.skills.title') }}
        </h3>
      </div>
      <Button
        size="sm"
        @click="handleCreate"
      >
        <FontAwesomeIcon
          :icon="['fas', 'plus']"
          class="mr-2"
        />
        {{ $t('bots.skills.addSkill') }}
      </Button>
    </div>

    <!-- Loading State -->
    <div
      v-if="isLoading"
      class="flex items-center justify-center py-8 text-sm text-muted-foreground"
    >
      <Spinner class="mr-2" />
      {{ $t('common.loading') }}
    </div>

    <!-- Empty State -->
    <div
      v-else-if="!skills.length"
      class="flex flex-col items-center justify-center py-12 text-center"
    >
      <div class="rounded-full bg-muted p-3 mb-4">
        <FontAwesomeIcon
          :icon="['fas', 'bolt']"
          class="size-6 text-muted-foreground"
        />
      </div>
      <h3 class="text-lg font-medium">
        {{ $t('bots.skills.emptyTitle') }}
      </h3>
      <p class="text-sm text-muted-foreground mt-1">
        {{ $t('bots.skills.emptyDescription') }}
      </p>
    </div>

    <!-- Skills Grid -->
    <div
      v-else
      class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"
    >
      <Card
        v-for="skill in skills"
        :key="skill.name"
        class="flex flex-col"
      >
        <CardHeader class="pb-3">
          <div class="flex items-start justify-between gap-2">
            <CardTitle
              class="text-base truncate"
              :title="skill.name"
            >
              {{ skill.name }}
            </CardTitle>
            <div class="flex items-center gap-1 shrink-0">
              <Button
                variant="ghost"
                size="sm"
                class="size-8 p-0"
                :title="$t('common.edit')"
                @click="handleEdit(skill)"
              >
                <FontAwesomeIcon
                  :icon="['fas', 'pen-to-square']"
                  class="size-3.5"
                />
              </Button>
              <ConfirmPopover
                :message="$t('bots.skills.deleteConfirm')"
                :loading="isDeleting && deletingName === skill.name"
                @confirm="handleDelete(skill.name)"
              >
                <template #trigger>
                  <Button
                    variant="ghost"
                    size="sm"
                    class="size-8 p-0 text-destructive hover:text-destructive"
                    :disabled="isDeleting"
                    :title="$t('common.delete')"
                  >
                    <FontAwesomeIcon
                      :icon="['fas', 'trash']"
                      class="size-3.5"
                    />
                  </Button>
                </template>
              </ConfirmPopover>
            </div>
          </div>
          <CardDescription
            class="line-clamp-2"
            :title="skill.description"
          >
            {{ skill.description || '-' }}
          </CardDescription>
        </CardHeader>
        <CardContent class="pb-4 grow">
          <div class="rounded-md bg-muted p-2 text-xs font-mono text-muted-foreground line-clamp-4 break-all">
            {{ skill.content }}
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- Edit Dialog -->
    <Dialog v-model:open="isDialogOpen">
      <DialogContent class="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{{ isEditing ? $t('common.edit') : $t('bots.skills.addSkill') }}</DialogTitle>
        </DialogHeader>
        <div class="space-y-4 py-4">
          <div class="space-y-2">
            <Label>{{ $t('common.name') }}</Label>
            <Input
              v-model="draftSkill.name"
              :placeholder="$t('common.namePlaceholder')"
              :disabled="isEditing || isSaving"
            />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('bots.skills.description') }}</Label>
            <Input
              v-model="draftSkill.description"
              :placeholder="$t('bots.skills.descriptionPlaceholder')"
              :disabled="isSaving"
            />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('bots.skills.content') }}</Label>
            <Textarea
              v-model="draftSkill.content"
              :placeholder="$t('bots.skills.contentPlaceholder')"
              :disabled="isSaving"
              class="min-h-[150px] font-mono text-sm"
            />
          </div>
        </div>
        <DialogFooter>
          <DialogClose as-child>
            <Button
              variant="outline"
              :disabled="isSaving"
            >
              {{ $t('common.cancel') }}
            </Button>
          </DialogClose>
          <Button
            :disabled="!canSave || isSaving"
            @click="handleSave"
          >
            <Spinner
              v-if="isSaving"
              class="mr-2 size-4"
            />
            {{ $t('common.save') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import {
  Button, Card, CardHeader, CardTitle, CardDescription, CardContent,
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogClose,
  Input, Textarea, Label, Spinner,
} from '@memoh/ui'
import ConfirmPopover from '@/components/confirm-popover/index.vue'
import {
  getBotsByBotIdContainerSkills,
  postBotsByBotIdContainerSkills,
  deleteBotsByBotIdContainerSkills,
  type HandlersSkillItem,
} from '@memoh/sdk'
import { resolveApiErrorMessage } from '@/utils/api-error'

const props = defineProps<{
  botId: string
}>()

const { t } = useI18n()

const isLoading = ref(false)
const isSaving = ref(false)
const isDeleting = ref(false)
const deletingName = ref('')
const skills = ref<HandlersSkillItem[]>([])

const isDialogOpen = ref(false)
const isEditing = ref(false)
const draftSkill = ref<HandlersSkillItem>({
  name: '',
  description: '',
  content: '',
})

const canSave = computed(() => {
  return (draftSkill.value.name || '').trim() && (draftSkill.value.content || '').trim()
})

async function fetchSkills() {
  if (!props.botId) return
  isLoading.value = true
  try {
    const { data } = await getBotsByBotIdContainerSkills({
      path: { bot_id: props.botId },
      throwOnError: true,
    })
    skills.value = data.skills || []
  } catch (error) {
    toast.error(resolveApiErrorMessage(error, t('bots.skills.loadFailed')))
  } finally {
    isLoading.value = false
  }
}

function handleCreate() {
  isEditing.value = false
  draftSkill.value = {
    name: '',
    description: '',
    content: '',
  }
  isDialogOpen.value = true
}

function handleEdit(skill: HandlersSkillItem) {
  isEditing.value = true
  draftSkill.value = {
    name: skill.name || '',
    description: skill.description || '',
    content: skill.content || '',
    metadata: skill.metadata,
  }
  isDialogOpen.value = true
}

async function handleSave() {
  if (!canSave.value) return
  isSaving.value = true
  try {
    await postBotsByBotIdContainerSkills({
      path: { bot_id: props.botId },
      body: {
        skills: [{
          name: draftSkill.value.name?.trim(),
          description: draftSkill.value.description?.trim(),
          content: draftSkill.value.content?.trim(),
          metadata: draftSkill.value.metadata,
        }],
      },
      throwOnError: true,
    })
    toast.success(t('bots.skills.saveSuccess'))
    isDialogOpen.value = false
    await fetchSkills()
  } catch (error) {
    toast.error(resolveApiErrorMessage(error, t('bots.skills.saveFailed')))
  } finally {
    isSaving.value = false
  }
}

async function handleDelete(name?: string) {
  if (!name) return
  isDeleting.value = true
  deletingName.value = name
  try {
    await deleteBotsByBotIdContainerSkills({
      path: { bot_id: props.botId },
      body: {
        names: [name],
      },
      throwOnError: true,
    })
    toast.success(t('bots.skills.deleteSuccess'))
    await fetchSkills()
  } catch (error) {
    toast.error(resolveApiErrorMessage(error, t('bots.skills.deleteFailed')))
  } finally {
    isDeleting.value = false
    deletingName.value = ''
  }
}

onMounted(() => {
  fetchSkills()
})
</script>
