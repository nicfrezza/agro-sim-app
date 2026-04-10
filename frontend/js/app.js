// ==========================================
// MQTT CLIENT (incluído aqui para evitar 404)
// ==========================================

class MQTTClient {
    constructor(onMessage) {
        this.client = null;
        this.onMessage = onMessage;
        this.messageCount = 0;
        this.lastCountTime = Date.now();
        this.connected = false;
    }

    connect() {
        try {
            const broker = 'ws://localhost:9002';

            if (typeof mqtt === 'undefined') {
                console.warn('MQTT library não carregada');
                this.updateStatus('⚠️ MQTT não disponível');
                return;
            }

            this.client = mqtt.connect(broker, {
                clientId: 'agro-web-' + Math.random().toString(16).substr(2, 8),
                clean: true,
                connectTimeout: 4000,
                reconnectPeriod: 1000,
            });

            this.client.on('connect', () => {
                console.log('✅ MQTT conectado');
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
                try {
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
                } catch (e) {
                    console.error('Erro ao parsear MQTT:', e);
                }
            });

            this.client.on('error', (err) => {
                console.error('❌ MQTT erro:', err);
                this.updateStatus('🔴 Erro');
            });

            this.client.on('offline', () => {
                this.connected = false;
                this.updateStatus('🟡 Offline');
            });

        } catch (err) {
            console.error('Falha ao conectar MQTT:', err);
            this.updateStatus('🔴 Falha');
        }
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
        const shortTopic = topic.replace('agro/sensors/', '').replace('/data', '').substring(0, 15);

        entry.innerHTML = `
            <span class="time">[${time}]</span>
            <span class="topic">${shortTopic}</span>:
            <span class="value">${data.value?.toFixed(1) || '--'}${data.unit || ''}</span>
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

// ==========================================
// AGRO SIM APP
// ==========================================

class AgroSimApp {
    constructor() {
        this.scene = null;
        this.camera = null;
        this.renderer = null;
        this.controls = null;
        this.terrain = null;
        this.raycaster = new THREE.Raycaster();
        this.mouse = new THREE.Vector2();
        this.sensors = new Map();
        this.selectedSensor = null;
        this.isDragging = false;
        this.mqttClient = null;
        this.ws = null;

        this.init();
    }

    init() {
        console.log('🚀 Iniciando AgroSim...');

        // Cena
        this.scene = new THREE.Scene();
        this.scene.background = new THREE.Color(0x87CEEB);
        this.scene.fog = new THREE.Fog(0x87CEEB, 50, 200);

        // Câmera
        this.camera = new THREE.PerspectiveCamera(
            75,
            window.innerWidth / window.innerHeight,
            0.1,
            1000
        );
        this.camera.position.set(0, 30, 50);

        // Renderer
        this.renderer = new THREE.WebGLRenderer({ antialias: true });
        this.renderer.setSize(window.innerWidth, window.innerHeight);
        this.renderer.shadowMap.enabled = true;
        this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
        document.body.appendChild(this.renderer.domElement);

        // Controles (com fallback)
        this.setupControls();

        // Luzes
        this.setupLights();

        // Terreno
        this.createTerrain();

        // Eventos
        this.setupEvents();

        // Conexões
        this.connectWebSocket();
        this.mqttClient = new MQTTClient();
        this.mqttClient.connect();

        // Loop
        this.animate();

        console.log('✅ AgroSim iniciado com sucesso!');
    }

    setupControls() {
        // Tenta usar OrbitControls se disponível
        if (typeof THREE.OrbitControls !== 'undefined') {
            console.log('Usando OrbitControls');
            this.controls = new THREE.OrbitControls(this.camera, this.renderer.domElement);
            this.controls.enableDamping = true;
            this.controls.dampingFactor = 0.05;
            this.controls.maxPolarAngle = Math.PI / 2 - 0.1;
            this.controls.minDistance = 10;
            this.controls.maxDistance = 150;
        } else {
            console.log('Usando controles básicos');
            this.setupBasicControls();
        }
    }

    setupBasicControls() {
        let isDragging = false;
        let previousMousePosition = { x: 0, y: 0 };
        const canvas = this.renderer.domElement;

        canvas.addEventListener('mousedown', (e) => {
            if (e.target !== canvas) return;
            isDragging = true;
            previousMousePosition = { x: e.clientX, y: e.clientY };
        });

        window.addEventListener('mouseup', () => {
            isDragging = false;
        });

        window.addEventListener('mousemove', (e) => {
            // Atualiza mouse para raycaster
            this.mouse.x = (e.clientX / window.innerWidth) * 2 - 1;
            this.mouse.y = -(e.clientY / window.innerHeight) * 2 + 1;

            // Controles de câmera
            if (!isDragging) return;

            const deltaX = e.clientX - previousMousePosition.x;
            const deltaY = e.clientY - previousMousePosition.y;

            // Rotaciona ao redor do centro
            const spherical = new THREE.Spherical();
            spherical.setFromVector3(this.camera.position);
            spherical.theta -= deltaX * 0.01;
            spherical.phi += deltaY * 0.01;
            spherical.phi = Math.max(0.1, Math.min(Math.PI / 2 - 0.1, spherical.phi));

            this.camera.position.setFromSpherical(spherical);
            this.camera.lookAt(0, 0, 0);

            previousMousePosition = { x: e.clientX, y: e.clientY };
        });

        // Zoom
        canvas.addEventListener('wheel', (e) => {
            const scale = e.deltaY > 0 ? 1.1 : 0.9;
            this.camera.position.multiplyScalar(scale);
        });
    }

    setupLights() {
        const ambientLight = new THREE.AmbientLight(0x404040, 0.5);
        this.scene.add(ambientLight);

        const sunLight = new THREE.DirectionalLight(0xffffff, 1);
        sunLight.position.set(50, 100, 50);
        sunLight.castShadow = true;
        sunLight.shadow.mapSize.width = 2048;
        sunLight.shadow.mapSize.height = 2048;
        sunLight.shadow.camera.left = -60;
        sunLight.shadow.camera.right = 60;
        sunLight.shadow.camera.top = 60;
        sunLight.shadow.camera.bottom = -60;
        this.scene.add(sunLight);
        this.sunLight = sunLight;

        // Visual do sol
        const sunGeo = new THREE.SphereGeometry(3, 16, 16);
        const sunMat = new THREE.MeshBasicMaterial({ color: 0xffff00 });
        const sunMesh = new THREE.Mesh(sunGeo, sunMat);
        sunMesh.position.copy(sunLight.position);
        this.scene.add(sunMesh);
    }

    createTerrain() {
        console.log('🌍 Criando terreno...');

        const geometry = new THREE.PlaneGeometry(100, 100, 64, 64);

        // Elevação procedural
        const positions = geometry.attributes.position;
        for (let i = 0; i < positions.count; i++) {
            const x = positions.getX(i);
            const y = positions.getY(i);
            const z = Math.sin(x * 0.08) * 4 + Math.cos(y * 0.08) * 4 +
                Math.sin(x * 0.2 + y * 0.15) * 1.5;
            positions.setZ(i, z);
        }

        geometry.computeVertexNormals();

        const material = new THREE.MeshStandardMaterial({
            color: 0x5d4037,
            roughness: 0.9,
            metalness: 0
        });

        this.terrain = new THREE.Mesh(geometry, material);
        this.terrain.rotation.x = -Math.PI / 2;
        this.terrain.name = 'terrain';
        this.terrain.receiveShadow = true;
        this.scene.add(this.terrain);

        // Grid
        const grid = new THREE.GridHelper(100, 50, 0x444444, 0x222222);
        grid.position.y = 0.1;
        this.scene.add(grid);

        // Água (rio)
        const waterGeo = new THREE.PlaneGeometry(15, 80);
        const waterMat = new THREE.MeshStandardMaterial({
            color: 0x006994,
            transparent: true,
            opacity: 0.7,
            roughness: 0.1
        });
        const water = new THREE.Mesh(waterGeo, waterMat);
        water.rotation.x = -Math.PI / 2;
        water.position.set(-35, 0.3, 0);
        this.scene.add(water);

        console.log('✅ Terreno criado!');
    }

    setupEvents() {
        window.addEventListener('resize', () => {
            this.camera.aspect = window.innerWidth / window.innerHeight;
            this.camera.updateProjectionMatrix();
            this.renderer.setSize(window.innerWidth, window.innerHeight);
        });

        const canvas = this.renderer.domElement;
        canvas.addEventListener('mousedown', (e) => this.onMouseDown(e));
        canvas.addEventListener('mouseup', () => this.onMouseUp());
    }

    onMouseDown(event) {
        this.raycaster.setFromCamera(this.mouse, this.camera);
        const intersects = this.raycaster.intersectObjects(this.scene.children, true);

        for (let intersect of intersects) {
            let obj = intersect.object;
            while (obj && !obj.userData.isSensor) {
                obj = obj.parent;
            }

            if (obj && obj.userData.isSensor) {
                this.selectedSensor = obj;
                this.isDragging = true;
                document.body.style.cursor = 'move';
                console.log('🎯 Sensor selecionado:', obj.userData.name);
                break;
            }
        }
    }

    onMouseUp() {
        if (this.isDragging) {
            this.isDragging = false;
            this.selectedSensor = null;
            document.body.style.cursor = 'default';
        }
    }

    // ============ API ============

    async addSensor(type) {
        const name = `${type}_${Math.floor(Math.random() * 1000)}`;
        const position = {
            x: (Math.random() - 0.5) * 40,
            y: 2,
            z: (Math.random() - 0.5) * 40
        };

        try {
            const response = await fetch('/api/sensors', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ type, position, name })
            });

            const sensor = await response.json();
            this.createSensorMesh(sensor);
            this.updateSensorList();
            console.log('✅ Sensor criado:', sensor);
        } catch (err) {
            console.error('❌ Erro ao criar sensor:', err);
            // Modo offline
            const offlineSensor = {
                id: 'local_' + Date.now(),
                type, position, name, active: false
            };
            this.createSensorMesh(offlineSensor);
            this.updateSensorList();
        }
    }

    createSensorMesh(sensor) {
        const configs = {
            temperature: { geo: [0.6, 1.5, 0.6], color: 0xff5722, y: 1.5 },
            humidity: { geo: [0.5, 16, 16], color: 0x2196f3, type: 'sphere' },
            rain: { geo: [0.6, 1.5, 16], color: 0x9c27b0, type: 'cone' },
            soil_moisture: { geo: [0.25, 0.25, 2, 16], color: 0x4caf50, type: 'cylinder' },
            wind: { geo: [0.15, 0.15, 2.5, 8], color: 0x00bcd4, type: 'cylinder' },
            solar: { geo: [1.2, 0.1, 0.8], color: 0xffeb3b, y: 0.5 }
        };

        const cfg = configs[sensor.type] || configs.temperature;
        let geometry;

        if (cfg.type === 'sphere') {
            geometry = new THREE.SphereGeometry(cfg.geo[0], cfg.geo[1], cfg.geo[2]);
        } else if (cfg.type === 'cone') {
            geometry = new THREE.ConeGeometry(cfg.geo[0], cfg.geo[1], cfg.geo[2]);
        } else if (cfg.type === 'cylinder') {
            geometry = new THREE.CylinderGeometry(cfg.geo[0], cfg.geo[1], cfg.geo[2], cfg.geo[3]);
        } else {
            geometry = new THREE.BoxGeometry(cfg.geo[0], cfg.geo[1], cfg.geo[2]);
        }

        const material = new THREE.MeshStandardMaterial({
            color: cfg.color,
            emissive: cfg.color,
            emissiveIntensity: 0.3,
            roughness: 0.4
        });

        const mesh = new THREE.Mesh(geometry, material);
        mesh.position.set(sensor.position.x, sensor.position.y || cfg.y || 1, sensor.position.z);
        mesh.castShadow = true;
        mesh.userData = {
            isSensor: true,
            id: sensor.id,
            type: sensor.type,
            name: sensor.name,
            active: sensor.active || false
        };

        // Base
        const base = new THREE.Mesh(
            new THREE.CylinderGeometry(0.15, 0.2, 0.4, 8),
            new THREE.MeshStandardMaterial({ color: 0x333333 })
        );
        base.position.y = -(cfg.y || 1) / 2 - 0.2;
        mesh.add(base);

        // LED
        const led = new THREE.Mesh(
            new THREE.SphereGeometry(0.12, 8, 8),
            new THREE.MeshBasicMaterial({
                color: sensor.active ? 0x00ff00 : 0x333333
            })
        );
        led.position.y = (cfg.y || 1) / 2 + 0.15;
        led.name = 'statusLed';
        mesh.add(led);

        // Label
        this.createLabel(mesh, sensor.name);

        this.scene.add(mesh);
        this.sensors.set(sensor.id, mesh);

        // Animação entrada
        mesh.scale.set(0, 0, 0);
        let s = 0;
        const grow = setInterval(() => {
            s += 0.1;
            mesh.scale.set(s, s, s);
            if (s >= 1) clearInterval(grow);
        }, 20);
    }

    createLabel(parent, text) {
        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');
        canvas.width = 256;
        canvas.height = 64;
        ctx.fillStyle = 'rgba(0,0,0,0.7)';
        ctx.fillRect(0, 0, 256, 64);
        ctx.fillStyle = 'white';
        ctx.font = 'bold 16px Arial';
        ctx.textAlign = 'center';
        ctx.fillText(text, 128, 40);

        const sprite = new THREE.Sprite(
            new THREE.SpriteMaterial({ map: new THREE.CanvasTexture(canvas) })
        );
        sprite.position.y = 3;
        sprite.scale.set(3, 0.75, 1);
        parent.add(sprite);
    }

    async updateSensorPosition(id, pos) {
        if (id.startsWith('local_')) return;
        try {
            await fetch(`/api/sensors/${id}/move`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(pos)
            });
        } catch (err) {
            console.error('Erro ao mover:', err);
        }
    }

    async toggleSensor(id) {
        const mesh = this.sensors.get(id);
        if (!mesh) return;

        let newState = !mesh.userData.active;

        if (!id.startsWith('local_')) {
            try {
                const res = await fetch(`/api/sensors/${id}/toggle`, { method: 'POST' });
                const data = await res.json();
                newState = data.active;
            } catch (err) {
                console.error('Erro toggle:', err);
            }
        }

        mesh.userData.active = newState;
        const led = mesh.getObjectByName('statusLed');
        if (led) led.material.color.setHex(newState ? 0x00ff00 : 0x333333);

        this.updateSensorList();
    }

    async deleteSensor(id) {
        if (!id.startsWith('local_')) {
            try { await fetch(`/api/sensors/${id}`, { method: 'DELETE' }); }
            catch (err) { console.error('Erro delete:', err); }
        }

        const mesh = this.sensors.get(id);
        if (mesh) {
            this.scene.remove(mesh);
            this.sensors.delete(id);
            this.updateSensorList();
        }
    }

    updateSensorList() {
        const list = document.getElementById('active-sensors');
        if (!list) return;

        if (this.sensors.size === 0) {
            list.innerHTML = '<li style="color: #888;">Nenhum sensor</li>';
            return;
        }

        list.innerHTML = '';
        const emojis = {
            temperature: '🌡️', humidity: '💧', rain: '🌧️',
            soil_moisture: '🌱', wind: '💨', solar: '☀️'
        };

        this.sensors.forEach((mesh, id) => {
            const li = document.createElement('li');
            const active = mesh.userData.active;
            li.innerHTML = `
                <div style="display:flex; justify-content:space-between; width:100%;">
                    <span>${emojis[mesh.userData.type] || '📡'} ${mesh.userData.name}</span>
                    <span class="status ${active ? 'active' : 'inactive'}">${active ? 'ON' : 'OFF'}</span>
                </div>
                <div style="width:100%; margin-top:5px;">
                    <button onclick="window.app.toggleSensor('${id}')" style="width:48%; margin-right:4%; padding:4px;">
                        ${active ? 'Desligar' : 'Ligar'}
                    </button>
                    <button onclick="window.app.deleteSensor('${id}')" style="width:48%; background:#f44336; padding:4px;">
                        🗑️
                    </button>
                </div>
            `;
            list.appendChild(li);
        });
    }

    // ============ WEBSOCKET ============

    connectWebSocket() {
        const ws = new WebSocket(`ws://${window.location.host}/ws`);

        ws.onopen = () => console.log('✅ WebSocket conectado');

        ws.onmessage = (e) => {
            try {
                const data = JSON.parse(e.data);
                this.handleWSMessage(data);
            } catch (err) {
                console.error('Erro WS:', err);
            }
        };

        ws.onclose = () => {
            console.log('WS fechado, reconectando...');
            setTimeout(() => this.connectWebSocket(), 3000);
        };

        this.ws = ws;
    }

    handleWSMessage(data) {
        switch (data.type) {
            case 'init':
                if (data.sensors) data.sensors.forEach(s => this.createSensorMesh(s));
                if (data.climate) this.updateClimate(data.climate);
                break;
            case 'climate':
                this.updateClimate(data.climate);
                break;
            case 'reading':
                this.handleReading(data.reading);
                break;
        }
    }

    updateClimate(c) {
        const ids = ['temp', 'humidity', 'rain', 'wind', 'solar'];
        const vals = [
            c.temperature?.toFixed(1),
            c.humidity?.toFixed(1),
            c.rain_intensity?.toFixed(1),
            c.wind_speed?.toFixed(1),
            c.solar_radiation?.toFixed(1)
        ];

        ids.forEach((id, i) => {
            const el = document.getElementById(`${id}-value`);
            if (el) el.textContent = vals[i] || '--';
        });

        // Cor do céu
        if (c.solar_radiation !== undefined) {
            const intensity = Math.max(0, Math.min(1, c.solar_radiation / 1000));
            const day = new THREE.Color(0x87CEEB);
            const sunset = new THREE.Color(0xff6b35);
            const night = new THREE.Color(0x1a1a2e);

            let sky;
            if (intensity > 0.5) {
                sky = sunset.clone().lerp(day, (intensity - 0.5) * 2);
            } else {
                sky = night.clone().lerp(sunset, intensity * 2);
            }

            this.scene.background = sky;
            this.scene.fog.color = sky;

            if (this.sunLight) this.sunLight.intensity = intensity;
        }
    }

    handleReading(r) {
        const mesh = this.sensors.get(r.sensor_id);
        if (!mesh || !mesh.userData.active) return;

        const led = mesh.getObjectByName('statusLed');
        if (led) {
            led.material.color.setHex(0xffffff);
            setTimeout(() => led.material.color.setHex(0x00ff00), 150);
        }
    }

    // ============ API CLIMA ============

    async setSeason(season) {
        try {
            await fetch('/api/climate/season', {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ season })
            });
            console.log('🌍 Estação:', season);
        } catch (err) {
            console.error('Erro estação:', err);
        }
    }

    async forceRain() {
        try {
            await fetch('/api/climate/rain', { method: 'POST' });
            console.log('🌧️ Chuva forçada!');
        } catch (err) {
            console.error('Erro chuva:', err);
        }
    }

    // ============ LOOP ============

    animate() {
        requestAnimationFrame(() => this.animate());

        if (this.controls && this.controls.update) {
            this.controls.update();
        }

        const time = Date.now() * 0.001;
        this.sensors.forEach(mesh => {
            if (mesh.userData.active) {
                mesh.rotation.y = Math.sin(time + mesh.id) * 0.2;
                mesh.material.emissiveIntensity = 0.3 + Math.sin(time * 2) * 0.2;
            }
        });

        this.renderer.render(this.scene, this.camera);
    }
}

// ==========================================
// INICIALIZAÇÃO GLOBAL
// ==========================================

let app;

document.addEventListener('DOMContentLoaded', () => {
    console.log('🎯 DOM pronto, criando app...');
    app = new AgroSimApp();
    window.app = app; // Expõe globalmente para os botões HTML
    console.log('✅ App disponível em window.app');
});