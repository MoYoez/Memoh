<template>
  <div class="flex gap-4 items-start">
    <div class=" p-2 rounded-full bg-[#F9F9F9]">
      <svg-icon
        type="mdi"
        :path="mdiRobotOutline"
      />
    </div>
    <section class="w-[90%]">
      <sup class="font-semibold">
        {{ robotSay.type }}
      </sup>
      <p class="leading-7 text-muted-foreground break-all">
        <template v-if="robotSay.state==='thinking'">
          <img
            src="data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIyNCIgaGVpZ2h0PSIyNCIgdmlld0JveD0iMCAwIDI0IDI0Ij48Y2lyY2xlIGN4PSI0IiBjeT0iMTIiIHI9IjMiIGZpbGw9ImN1cnJlbnRDb2xvciI+PGFuaW1hdGUgaWQ9IlNWRzlJZ2JSYnNsIiBhdHRyaWJ1dGVOYW1lPSJyIiBiZWdpbj0iMDtTVkdGVU5wQ1dkRy5lbmQtMC4yNXMiIGR1cj0iMC43NXMiIHZhbHVlcz0iMzsuMjszIi8+PC9jaXJjbGU+PGNpcmNsZSBjeD0iMTIiIGN5PSIxMiIgcj0iMyIgZmlsbD0iY3VycmVudENvbG9yIj48YW5pbWF0ZSBhdHRyaWJ1dGVOYW1lPSJyIiBiZWdpbj0iU1ZHOUlnYlJic2wuZW5kLTAuNnMiIGR1cj0iMC43NXMiIHZhbHVlcz0iMzsuMjszIi8+PC9jaXJjbGU+PGNpcmNsZSBjeD0iMjAiIGN5PSIxMiIgcj0iMyIgZmlsbD0iY3VycmVudENvbG9yIj48YW5pbWF0ZSBpZD0iU1ZHRlVOcENXZEciIGF0dHJpYnV0ZU5hbWU9InIiIGJlZ2luPSJTVkc5SWdiUmJzbC5lbmQtMC40NXMiIGR1cj0iMC43NXMiIHZhbHVlcz0iMzsuMjszIi8+PC9jaXJjbGU+PC9zdmc+"
            class="inline"
            alt="thinking"
          >
        </template>
        <template v-else>
          <MarkdownRender
            :content="robotSay.description"
            custom-id="chat-answer"
          />
        </template>
      </p>
    </section>
  </div>
</template>

<script setup lang="ts">
import SvgIcon from '@jamescoyle/vue-icon'
import { mdiRobotOutline } from '@mdi/js'
import type { robot } from '@memoh/shared'
import MarkdownRender,{enableKatex,enableMermaid} from 'markstream-vue'
enableKatex()
enableMermaid()
const {robotSay}=defineProps<{
  robotSay: robot
}>()

</script>