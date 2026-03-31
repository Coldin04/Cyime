import * as Y from 'yjs';
import { WebsocketProvider } from 'y-websocket';
import type { Awareness } from 'y-protocols/awareness';

interface ProviderConfig {
  wsUrl: string;
  documentId: string;
  userId: string;
  token: string;
}

interface ProviderInstance {
  provider: WebsocketProvider | null;
  doc: Y.Doc | null;
  awareness: Awareness | null;
  isConnected: boolean;
  error: string | null;
}

class YjsProviderManager {
  private instances = new Map<string, ProviderInstance>();
  private connectionTimeouts = new Map<string, NodeJS.Timeout>();
  private readonly CONNECTION_TIMEOUT = 10000; // 10s timeout

  /**
   * 创建或获取 Y.js 协作 provider
   */
  async createProvider(config: ProviderConfig): Promise<ProviderInstance> {
    const docId = config.documentId;

    // 检查是否已有实例
    if (this.instances.has(docId)) {
      return this.instances.get(docId)!;
    }

    const instance: ProviderInstance = {
      provider: null,
      doc: null,
      awareness: null,
      isConnected: false,
      error: null
    };

    try {
      // 构建 WebSocket URL
      const wsUrl = this.buildWebSocketUrl(config.wsUrl, config.documentId, config.token);

      const ydoc = new Y.Doc();

      // 创建 WebSocket provider
      const provider = new WebsocketProvider(wsUrl, `doc:${docId}`, ydoc);

      // 设置连接超时检测
      await this.waitForConnection(provider, this.CONNECTION_TIMEOUT);

      instance.provider = provider;
      instance.doc = ydoc;
      instance.awareness = provider.awareness;
      instance.isConnected = true;

      // 监听连接状态变化
      this.setupConnectionHandlers(docId, provider, instance);

      // 保存到实例缓存
      this.instances.set(docId, instance);

      console.log(`[Yjs] Provider created for document ${docId}`);
      return instance;
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : 'Unknown error';
      instance.error = errorMsg;
      instance.isConnected = false;

      console.warn(
        `[Yjs] Failed to create WebSocket provider for ${docId}: ${errorMsg}. Falling back to local mode.`
      );

      // 降级：仍然创建本地 Y.js doc，但不连接 WS
      try {
        const ydoc = new Y.Doc();
        instance.doc = ydoc;
      } catch (fallbackError) {
        console.error('[Yjs] Fallback failed:', fallbackError);
      }

      this.instances.set(docId, instance);
      return instance;
    }
  }

  /**
   * 等待 WebSocket 连接建立
   */
  private async waitForConnection(
    provider: WebsocketProvider,
    timeout: number
  ): Promise<void> {
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        reject(new Error('WebSocket connection timeout'));
      }, timeout);

      const checkConnection = () => {
        // 检查内部 ws 对象的连接状态
        if ((provider as any).ws?.readyState === 1) {
          clearTimeout(timer);
          resolve();
        } else {
          setTimeout(checkConnection, 100);
        }
      };

      checkConnection();
    });
  }

  /**
   * 设置连接状态处理器
   */
  private setupConnectionHandlers(
    docId: string,
    provider: WebsocketProvider,
    instance: ProviderInstance
  ): void {
    provider.on('status', (event: { status: 'connected' | 'disconnected' | 'connecting' }) => {
      instance.isConnected = event.status === 'connected';
      if (event.status === 'connected') {
        console.log(`[Yjs] Connected to collaboration server for ${docId}`);
        instance.error = null;
      } else if (event.status === 'disconnected') {
        console.warn(`[Yjs] Disconnected from collaboration server for ${docId}`);
      }
    });

    provider.on('connection-error', (event: Event, _provider: WebsocketProvider) => {
      console.error(`[Yjs] Connection error for ${docId}:`, event);
      instance.error = 'Connection error occurred';
      instance.isConnected = false;
    });

    provider.on('sync', (isSynced: boolean) => {
      if (isSynced) {
        console.log(`[Yjs] Document ${docId} synced`);
      }
    });
  }

  /**
   * 构建 WebSocket URL
   */
  private buildWebSocketUrl(
    baseUrl: string,
    documentId: string,
    token: string
  ): string {
    // baseUrl 来自配置，可能是相对路径或完整 URL
    let url = baseUrl;

    // 如果是相对路径，添加当前协议和主机
    if (
      !url.startsWith('http://') &&
      !url.startsWith('https://') &&
      !url.startsWith('ws://') &&
      !url.startsWith('wss://')
    ) {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const host = window.location.host;
      url = `${protocol}//${host}${url}`;
    }

    // 转换 http/https 为 ws/wss
    url = url.replace(/^https:/, 'wss:').replace(/^http:/, 'ws:');

    // 添加 token query 参数
    const separator = url.includes('?') ? '&' : '?';
    return `${url}${separator}token=${encodeURIComponent(token)}`;
  }

  /**
   * 获取现有 provider 实例
   */
  getProvider(documentId: string): ProviderInstance | undefined {
    return this.instances.get(documentId);
  }

  /**
   * 销毁 provider 实例
   */
  destroyProvider(documentId: string): void {
    const instance = this.instances.get(documentId);
    if (instance) {
      if (instance.provider) {
        instance.provider.destroy();
      }
      if (instance.doc) {
        instance.doc.destroy();
      }
      this.instances.delete(documentId);
      console.log(`[Yjs] Provider destroyed for document ${documentId}`);
    }

    // 清理超时计时器
    const timeout = this.connectionTimeouts.get(documentId);
    if (timeout) {
      clearTimeout(timeout);
      this.connectionTimeouts.delete(documentId);
    }
  }

  /**
   * 销毁所有实例
   */
  destroyAll(): void {
    for (const [docId] of this.instances) {
      this.destroyProvider(docId);
    }
  }
}

export const yjsProvider = new YjsProviderManager();
export type { ProviderConfig, ProviderInstance };