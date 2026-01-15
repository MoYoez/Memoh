/**
 * Basic usage examples for @memoh/container
 */

import { createContainer, useContainer, listContainers } from '../src'

async function main() {
  console.log('🚀 Container Management Examples\n')

  // Example 1: Create and start a container
  console.log('📦 Example 1: Create and start a container')
  try {
    const container = await createContainer({
      name: 'example-nginx',
      image: 'docker.io/library/nginx:alpine',
      env: {
        NGINX_HOST: 'localhost',
        NGINX_PORT: '80',
      },
      labels: {
        example: 'basic',
        version: '1.0',
      },
    })

    console.log('✅ Container created:', container.id)
    console.log('   Status:', container.status)
    console.log('   Image:', container.image)
    console.log('')

    // Start the container
    const ops = useContainer(container.name)
    await ops.start()
    console.log('✅ Container started\n')

    // Get container info
    const info = await ops.info()
    console.log('📊 Container info:')
    console.log('   Name:', info.name)
    console.log('   Status:', info.status)
    console.log('   Created:', info.createdAt)
    console.log('')

    // Stop and remove
    await ops.stop(5)
    console.log('⏹️  Container stopped')
    await ops.remove()
    console.log('🗑️  Container removed\n')
  } catch (error) {
    console.error('❌ Error:', error)
  }

  // Example 2: List all containers
  console.log('📋 Example 2: List all containers')
  try {
    const containers = await listContainers()
    if (containers.length === 0) {
      console.log('   No containers found\n')
    } else {
      for (const container of containers) {
        console.log(`   - ${container.name}: ${container.status}`)
      }
      console.log('')
    }
  } catch (error) {
    console.error('❌ Error:', error)
  }

  // Example 3: Execute commands in container
  console.log('🔧 Example 3: Execute commands in container')
  try {
    const container = await createContainer({
      name: 'example-alpine',
      image: 'docker.io/library/alpine:latest',
      command: ['sh', '-c', 'while true; do sleep 1; done'],
    })

    const ops = useContainer(container.name)
    await ops.start()
    console.log('✅ Container started')

    // Execute command
    const result = await ops.exec(['echo', 'Hello from container!'])
    console.log('📤 Command output:', result.stdout)
    console.log('   Exit code:', result.exitCode)
    console.log('')

    // Cleanup
    await ops.stop(2)
    await ops.remove()
    console.log('🧹 Cleaned up\n')
  } catch (error) {
    console.error('❌ Error:', error)
  }

  console.log('✨ All examples completed!')
}

// Run examples
main().catch(console.error)

