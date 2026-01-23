<template>
  <section>
    <AddPlatform />
    <div>
      <menu class="grid grid-cols-4 gap-4 [&_li>*]:h-full">
        <li>
          <Card>
            <CardHeader>
              <CardTitle class="text-muted-foreground">
                平台:Telegram
              </CardTitle>
              <CardContent class="mt-4 p-0">
                <ol
                  class=" [&>li]:mt-2"
                  type="1"
                >
                  <li>功能1</li>
                  <li>功能2</li>
                  <li>功能2</li>
                </ol>
              </CardContent>
            </CardHeader>
            <CardContent>
              <section class="flex" />
            </CardContent>
            <CardFooter class="flex gap-4">
              <Switch />
              <Button
                class="ml-auto"
                @click="open = true"
              >
                编辑
              </Button>
              <Button variant="destructive">
                删除
              </Button>
            </CardFooter>
          </Card>
        </li>
        <li>
          <Card>
            <CardHeader>
              <CardTitle class="text-muted-foreground">
                平台:Telegram
              </CardTitle>
              <CardContent class="mt-4 p-0">
                <ol
                  class=" [&>li]:mt-2"
                  type="1"
                >
                  <li>功能1</li>
                  <li>功能2</li>
                  <li>功能2</li>
                </ol>
              </CardContent>
            </CardHeader>
            <CardContent>
              <section class="flex" />
            </CardContent>
            <CardFooter class="flex gap-4">
              <Switch />
              <Button class="ml-auto">
                编辑
              </Button>
              <Button variant="destructive">
                删除
              </Button>
            </CardFooter>
          </Card>
        </li>
        <li>
          <Card>
            <CardHeader>
              <CardTitle class="text-muted-foreground">
                平台:Telegram
              </CardTitle>
              <CardContent class="mt-4 p-0">
                <ol
                  class=" [&>li]:mt-2"
                  type="1"
                >
                  <li>功能1</li>
                  <li>功能2</li>
                  <li>功能2</li>
                </ol>
              </CardContent>
            </CardHeader>
            <CardContent>
              <section class="flex" />
            </CardContent>
            <CardFooter class="flex gap-4">
              <Switch />
              <Button class="ml-auto">
                编辑
              </Button>
              <Button variant="destructive">
                删除
              </Button>
            </CardFooter>
          </Card>
        </li>
        <li>
          <Card>
            <CardHeader>
              <CardTitle class="text-muted-foreground">
                平台:Telegram
              </CardTitle>
              <CardContent class="mt-4 p-0">
                <ol
                  class=" [&>li]:mt-2"
                  type="1"
                >
                  <li>功能1</li>
                  <li>功能2</li>
                  <li>功能2</li>
                </ol>
              </CardContent>
            </CardHeader>
            <CardContent>
              <section class="flex" />
            </CardContent>
            <CardFooter class="flex gap-4">
              <Switch />
              <Button class="ml-auto">
                编辑
              </Button>
              <Button variant="destructive">
                删除
              </Button>
            </CardFooter>
          </Card>
        </li>
        <li>
          <Card>
            <CardHeader>
              <CardTitle class="text-muted-foreground flex justify-between">
                <header>
                  平台:Telegram
                </header>
                
                <Badge variant="outline">
                  运行中...
                </Badge>
              </CardTitle>
              <CardContent class="mt-4 p-0">
                <ol
                  class=" [&>li]:mt-2"
                  type="1"
                >
                  <li>功能1</li>
                  <li>功能2</li>
                  <li>功能2</li>
                </ol>
              </CardContent>
            </CardHeader>
            <CardContent>
              <section class="flex" />
            </CardContent>
            <CardFooter class="flex gap-4">
              <Switch />
              <Button
                class="ml-auto"
                @click="open=true"
              >
                编辑
              </Button>
              <Button variant="destructive">
                删除
              </Button>
            </CardFooter>
          </Card>
        </li>
        <li>
          <Card>
            <CardHeader>
              <CardTitle class="text-muted-foreground">
                平台:Telegram
              </CardTitle>
              <CardContent class="mt-4 p-0">
                <ol
                  class=" [&>li]:mt-2 "
                  type="1"
                >
                  <li>1. 功能1</li>
                  <li>2. 功能2</li>
                  <li>3. 功能2</li>
                </ol>
              </CardContent>
            </CardHeader>
            <CardContent>
              <section class="flex" />
            </CardContent>
            <CardFooter class="flex gap-4">
              <Switch />
              <Button class="ml-auto">
                编辑
              </Button>
              <Button variant="destructive">
                删除
              </Button>
            </CardFooter>
          </Card>
        </li>
      </menu>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useMutation, useQuery } from '@pinia/colada'
import request from '@/utils/request'
import { watch, h, provide, ref } from 'vue'
import AddPlatform from '@/components/AddPlatform/index.vue'
import {Card,CardHeader,CardFooter,CardContent,CardTitle,Switch,Button, Badge } from '@memoh/ui'

const open = ref(false)

provide('open', open)

const { data: platformData } = useQuery({
  key: ['platform'],
  query: () => request({
    url: '/platform/'
  })
})


watch(platformData, () => {
  console.log(platformData.value?.data)
})
</script>