/**
 * Containerd client implementation using ctr CLI
 */

import { execa } from 'execa'
import type { ContainerConfig, ContainerInfo, ContainerStatus, ContainerdOptions } from './types'

export const buildExecCommand = (name: string, command: string[]) => ['task', 'exec', '--exec-id', `exec-${Date.now()}`, name, ...command]

/**
 * Containerd client for managing containers
 */
export class ContainerdClient {
  private namespace: string
  private socket?: string
  private timeout: number

  constructor(options: ContainerdOptions = {}) {
    this.namespace = options.namespace || 'default'
    this.socket = options.socket
    this.timeout = options.timeout || 30000
  }

  /**
   * Build ctr command with options
   */
  private buildCtrCommand(args: string[]): string[] {
    const cmd = ['ctr']
    
    if (this.socket) {
      cmd.push('--address', this.socket)
    }
    
    cmd.push('--namespace', this.namespace)
    cmd.push(...args)
    
    return cmd
  }

  /**
   * Execute ctr command
   */
  private async exec(args: string[]): Promise<{ stdout: string; stderr: string }> {
    const cmd = this.buildCtrCommand(args)
    const [program, ...programArgs] = cmd
    
    try {
      const result = await execa(program, programArgs, {
        timeout: this.timeout,
      })
      
      return {
        stdout: result.stdout,
        stderr: result.stderr,
      }
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : String(error)
      throw new Error(`Containerd command failed: ${message}`)
    }
  }

  /**
   * Pull container image
   */
  async pullImage(image: string): Promise<void> {
    await this.exec(['image', 'pull', image])
  }

  /**
   * Create a new container
   */
  async createContainer(config: ContainerConfig): Promise<ContainerInfo> {
    const args = ['container', 'create']
    
    // Add image
    args.push(config.image)
    
    // Add container name
    args.push(config.name)
    
    // Add command if specified
    if (config.command && config.command.length > 0) {
      args.push(...config.command)
    }
    
    await this.exec(args)
    
    // Return container info
    return this.getContainerInfo(config.name)
  }

  /**
   * Start a container
   */
  async startContainer(name: string): Promise<void> {
    await this.exec(['task', 'start', '--detach', name])
  }

  /**
   * Stop a container
   */
  async stopContainer(name: string, timeout: number = 10): Promise<void> {
    try {
      await this.exec(['task', 'kill', '--signal', 'SIGTERM', name])
      
      // Wait for graceful shutdown
      await new Promise(resolve => setTimeout(resolve, timeout * 1000))
      
      // Force kill if still running
      try {
        await this.exec(['task', 'kill', '--signal', 'SIGKILL', name])
      } catch {
        // Container might have already stopped
      }
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : ''
      if (!message.includes('not found')) {
        throw error
      }
    }
  }

  /**
   * Pause a container
   */
  async pauseContainer(name: string): Promise<void> {
    await this.exec(['task', 'pause', name])
  }

  /**
   * Resume a paused container
   */
  async resumeContainer(name: string): Promise<void> {
    await this.exec(['task', 'resume', name])
  }

  /**
   * Remove a container
   */
  async removeContainer(name: string, force: boolean = false): Promise<void> {
    if (force) {
      // Try to stop the task first
      try {
        await this.exec(['task', 'kill', '--signal', 'SIGKILL', name])
        await this.exec(['task', 'delete', name])
      } catch {
        // Task might not exist
      }
    }
    
    await this.exec(['container', 'delete', name])
  }

  /**
   * Execute command in container
   */
  async execInContainer(name: string, command: string[]): Promise<{ stdout: string; stderr: string; exitCode: number }> {
    const args = buildExecCommand(name, command)
    
    try {
      const result = await this.exec(args)
      return {
        stdout: result.stdout,
        stderr: result.stderr,
        exitCode: 0,
      }
    } catch (error: unknown) {
      const err = error as { stdout?: string; stderr?: string; exitCode?: number; message?: string }
      return {
        stdout: err.stdout || '',
        stderr: err.stderr || err.message || '',
        exitCode: err.exitCode || 1,
      }
    }
  }

  /**
   * Get container information
   */
  async getContainerInfo(name: string): Promise<ContainerInfo> {
    const result = await this.exec(['container', 'info', name])
    
    try {
      const info = JSON.parse(result.stdout)
      
      return {
        id: info.ID || name,
        name: name,
        image: info.Image || '',
        status: await this.getContainerStatus(name),
        namespace: this.namespace,
        createdAt: info.CreatedAt ? new Date(info.CreatedAt) : new Date(),
        labels: info.Labels || {},
      }
    } catch {
      // Fallback if JSON parsing fails
      return {
        id: name,
        name: name,
        image: '',
        status: 'unknown',
        namespace: this.namespace,
        createdAt: new Date(),
      }
    }
  }

  /**
   * Get container status
   */
  async getContainerStatus(name: string): Promise<ContainerStatus> {
    try {
      const result = await this.exec(['task', 'list'])
      const lines = result.stdout.split('\n')
      
      for (const line of lines) {
        if (line.includes(name)) {
          if (line.includes('RUNNING')) return 'running'
          if (line.includes('PAUSED')) return 'paused'
          if (line.includes('STOPPED')) return 'stopped'
        }
      }
      
      // Container exists but no task
      return 'created'
    } catch {
      return 'unknown'
    }
  }

  /**
   * Get container logs
   */
  async getContainerLogs(name: string): Promise<string> {
    try {
      const result = await this.exec(['task', 'logs', name])
      return result.stdout
    } catch (error: unknown) {
      return error instanceof Error ? error.message : ''
    }
  }

  /**
   * List all containers
   */
  async listContainers(): Promise<ContainerInfo[]> {
    const result = await this.exec(['container', 'list', '--quiet'])
    const containerNames = result.stdout.split('\n').filter(name => name.trim())
    
    const containers: ContainerInfo[] = []
    for (const name of containerNames) {
      try {
        const info = await this.getContainerInfo(name)
        containers.push(info)
      } catch {
        // Skip containers that can't be accessed
      }
    }
    
    return containers
  }

  /**
   * Check if container exists
   */
  async containerExists(name: string): Promise<boolean> {
    try {
      await this.getContainerInfo(name)
      return true
    } catch {
      return false
    }
  }
}

