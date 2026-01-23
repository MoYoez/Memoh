<template>
  <section class="flex">
    <Dialog v-model:open="open ">
      <DialogTrigger as-child>
        <Button
          variant="default"
          class="ml-auto my-4"
        >
          添加平台
        </Button>
      </DialogTrigger>
      <DialogContent class="sm:max-w-106.25">
        <form @submit="addPlatform">
          <DialogHeader>
            <DialogTitle>添加平台</DialogTitle>
            <DialogDescription class="mb-4">
              为模型添加使用平台
            </DialogDescription>
          </DialogHeader>
          <div>
            <FormField
              v-slot="{ componentField }"
              name="name"
            >
              <FormItem>
                <FormLabel class="mb-2">
                  Name
                </FormLabel>
                <FormControl>
                  <Input
                    type="text"
                    placeholder="请输入Name"
                    v-bind="componentField"
                    autocomplete="name"
                  />
                </FormControl>
                <blockquote class="h-5">
                  <FormMessage />
                </blockquote>
              </FormItem>
            </FormField>
            <FormField
              v-slot="{ componentField }"
              name="config"
            >
              <FormItem>
                <FormLabel class="mb-2">
                  Config
                </FormLabel>
                <FormControl>
                  <TagsInput
                    :add-on-blur="true"
                    :model-value="configList"
                    :convert-value="tagStr => {
                      if (/^\w+\:\w+$/.test(tagStr)) {
                        return tagStr
                      }
                      return ''
                    }"
                    @update:model-value="(env) => {
                      configList = env.filter(Boolean) as string[]
                      const curConfig: Record<string,string> = {}
                      configList.forEach(envItem => {
                        const [key, value] = envItem.split(`:`);
                        if (key && value) {
                          curConfig[key] = value
                        }
                      })
                      componentField['onUpdate:modelValue']?.(curConfig)
                    }"
                  >
                    <TagsInputItem
                      v-for="(value, index) in configList"
                      :key="index"
                      :value="value as string"
                    >
                      <TagsInputItemText />
                      <TagsInputItemDelete />
                    </TagsInputItem>
                    <TagsInputInput
                      placeholder="请输入Env"
                      class="w-full py-1"
                    />
                  </TagsInput>
                </FormControl>
                <blockquote class="h-5">
                  <FormMessage />
                </blockquote>
              </FormItem>
            </FormField>
            <FormField
              v-slot="{ componentField }"
              name="active"
            >
              <FormItem>
                <FormLabel class="mb-2">
                  是否立即使用
                </FormLabel>
                <FormControl>
                  <Switch
                    id="airplane-mode"
                    :model-value="componentField.modelValue"
                    @update:model-value="componentField['onUpdate:modelValue']"
                  />
                </FormControl>
                <blockquote class="h-5">
                  <FormMessage />
                </blockquote>
              </FormItem>
            </FormField>
          </div>
          <DialogFooter class="mt-4">
            <DialogClose as-child>
              <Button variant="outline">
                Cancel
              </Button>
            </DialogClose>
            <Button type="submit">
              添加MCP
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </section>
</template>

<script setup lang="ts">
import {  
  Button,
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  Input,
  FormField,
  FormControl,
  FormItem,
  FormLabel,
  FormMessage,
  TagsInput,
  TagsInputInput,
  TagsInputItem,
  TagsInputItemDelete,
  TagsInputItemText,
  Switch,  
} from '@memoh/ui'
import z from 'zod'
import { toTypedSchema } from '@vee-validate/zod'
import { useForm } from 'vee-validate'
import { ref, inject } from 'vue'
import { useMutation } from '@pinia/colada'
import request from '@/utils/request'

const configList=ref<string[]>([])
const validataSchema = toTypedSchema(z.object({
  name: z.string().min(1),
  config: z.looseObject({}),
  active:z.coerce.boolean()
}))

const form = useForm({
  validationSchema:validataSchema
})


const {mutate:addFetchPlatform}=useMutation({
  mutation: (data: Parameters<(Parameters<typeof form.handleSubmit>)[0]>[0]) => request({
    url: '/platform/',
    data,
    method:'post'
  })
})
const addPlatform = form.handleSubmit(async (value) => {
  try {
    await addFetchPlatform(value)  
    open.value=false
  } catch {
    return
  }  
})

const open=inject('open',ref<boolean>(false))
</script>