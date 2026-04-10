class MQTTClient {
    constructor(onMessage) {
        this.client = null;
        this.onMessage = onMessage;
        this.messageCount = 0;
        this.lastCountTime = Date.now();
        this.connected = false;
    }

    connect() {
        // Usa porta 9002 (WebSocket)
        const broker = 'ws://localhost:9002';

        // mqtt está disponível globalmente agora
        this.client = mqtt.connect(broker, {
            clientId: 'agro-web-' + Math.random().toString(16).substr(2, 8),
            clean: true,
            connectTimeout: 4000,
            reconnectPeriod: 1000,
        });

        this.client.on('connect', () => {
            console.log('MQTT conectado');
            this.connected = true;
            this.updateStatus('🟢 Online');

            this.client.subscribe('agro/sensors/+/data', (err) => {
                if (!err) console.log('Subscrito em sensores');
            });

            this.client.subscribe('agro/climate/current', (err) => {
                if (!err) console.log('Subscrito em clima');
            });
        });

        this.client.on('message', (topic, payload) => {
            this.messageCount++;
            const data = JSON.parse(payload.toString());

            const now = Date.now();
            if (now - this.lastCountTime > 1000) {
                const rate = document.getElementById('msg-rate');
                if (rate) rate.textContent = this.messageCount;
                this.messageCount = 0;
                this.lastCountTime = now;
            }

            if (this.onMessage) this.onMessage(topic, data);
            this.logMessage(topic, data);
        });

        this.client.on('error', (err) => {
            console.error('MQTT erro:', err);
            this.updateStatus('🔴 Erro');
        });

        this.client.on('offline', () => {
            this.connected = false;
            this.updateStatus('🟡 Offline');
        });
    }

    updateStatus(status) {
        const el = document.getElementById('mqtt-status');
        if (el) el.textContent = status;
    }

    logMessage(topic, data) {
        const log = document.getElementById('mqtt-log');
        if (!log) return;

        const entry = document.createElement('div');
        entry.className = 'log-entry';

        const time = new Date().toLocaleTimeString();
        const shortTopic = topic.replace('agro/sensors/', '').replace('/data', '').substring(0, 20);

        entry.innerHTML = `
            <span class="time">[${time}]</span>
            <span class="topic">${shortTopic}</span>:
            <span class="value">${data.value?.toFixed(2) || '--'}${data.unit || ''}</span>
        `;

        log.insertBefore(entry, log.firstChild);
        if (log.children.length > 50) log.removeChild(log.lastChild);
    }

    publish(topic, payload) {
        if (this.client && this.connected) {
            this.client.publish(topic, JSON.stringify(payload));
        }
    }
}