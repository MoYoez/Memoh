import { defineStore } from 'pinia'
import { reactive, watch,ref} from 'vue'
import type { user, robot } from '@memoh/shared'
import loadRobotChat from '@/utils/loadRobotChat'
import str from '../../demo.md?raw'

export const useChatList= defineStore('chatList', () => {
  const chatList = reactive<(((user | robot)))[]>([])
  const loading=ref(false)
  const add = (chatItem: user | robot) => {
    chatList.push(chatItem)
  }
  // 监听状态的watch,同一时间只能有一个thinking和complete
  watch(chatList, () => {
    const robotType=chatList.filter(chatItem => chatItem.action === 'robot')
    const isLoading = robotType.some(robotItem => robotItem.state === 'thinking'||robotItem.state==='generate') 
    if (isLoading) {
      loading.value=true
    } else {
      loading.value=false
    }
    const generateItem = robotType.find(robotItem => robotItem.state === 'thinking')
    // 模拟一下改变状态
    setTimeout(() => {
      if (generateItem) {
        loadRobotChat(generateItem, str)
      }
    },3000)
  }, {
    immediate:true
  })
  return {
    chatList,
    add,
    loading
  }
})