import * as THREE from 'three';

export class Terrain {
    constructor(scene) {
        this.scene = scene;
        this.sensors = new Map();
        this.raycaster = new THREE.Raycaster();
        this.mouse = new THREE.Vector2();
        this.selectedSensor = null;
        this.isDragging = false;

        this.createTerrain();
        this.setupLighting();
        this.setupInteraction();
    }

    createTerrain() {
        // geometria do terrno com elevação 
        const geometry = new THREE.PlaneGeometry(100, 100, 64, 64);

        // adiciona elevação procedural (ruído simples)
        const positions = geometry.attributes.position;
        for (let i = 0; i < positions.count; i++) {
            const x = positions.getX(i);
            const y = positions.getY(i);
            //função de elevação: coluna suaves
            const z = Math.sin(x * 0.1) * 2 + Math.cos(y * 0.1) * 2 + Math.sin(x + 0.3 + y * 0.2) * 0.5;
            positions.setZ(i, z);

        }

        geometry.computeVertexNormals();

        //MATERIAL DO SOLO COM TEXTURA PROCEDURAL 
        const material = new THREE.MeshStandardMaterial({
            color: 0x556655,
            roughness: 0.8,
            metalness: 0.2,
            flatShading: false, // suaviza as normais para um visual mais natural
        });

        this.terrain = new THREE.Mesh(geometry, material);
        this.terrain.rotation.x = -Math.PI / 2;
        this.terrain.name = 'terrain';
        this.scene.add(this.terrain);


        // grid helper para referencia 
        const grid = new THREE.GridHelper(100, 50, 0x444444, 0x222222);
        grid.position.y = 0.01;
        this.scene.add(grid);

    }


    setupLighting() {
        //Sol 
        const sunLight = new THREE.DirectionalLight(0xffffff, 1);
        sunLight.position.set(50, 100, 50);
        sunLight.castShadow = true;
        sunLight.shadow.mapSize.width = 1024;
        sunLight.shadow.mapSize.height = 1024;
        this.scene.add(sunLight);

        // Ambient 
        const ambient = new THREE.AmbientLight(0x404040, 0.5);
        this.scene.add(ambient);
    }


    CreateSensor(type, position, name) {
        // geometria baseada no tipo 
        let geometry, color;
        switch (type) {
            case 'temperature':
                geometry = new THREE.BoxGeometry(1, 1, 1);
                color = 0xff5722; // laranja 
                break;
            case 'humidity':
                geometry = new THREE.SphereGeometry(0.8, 16, 16);
                color = 0x2196f3; // azul
                break;
            case 'rain':
                geometry = new THREE.ConeGeometry(0.5, 1, 16);
                color = 0x9c27b0; // Roxo
                break;
            case 'soil_moisture':
                geometry = new THREE.CylinderGeometry(0.5, 0.5, 1, 16);
                color = 0x4caf50; // Verde
                break;
            default:
                geometry = new THREE.BoxGeometry(1, 1, 1);
                color = 0x757575 // cinza
        }

        const material = new THREE.MeshStandardMaterial({
            color: color,
            emissive: color,
            emissiveIntensity: 0.2
        });

        const mesh = new THREE.Mesh(geometry, material); // cria o mesh do sensor // mesh é a representação visual do sensor no terreno
        mesh.position.copy(position);
        mesh.castShadow = true;
        mesh.userData = {
            isSensor: true,
            sensorType: type,
            id: null,  // será preenchido quando o sensor for adicionado ao mapa
            name: name,
            active: false
        };

        //base do sensor (ponto de contato com solo) 
        const baseGeo = new THREE.CylinderGeometry(0.3, 0.3, 0.2, 16);
        const baseMat = new THREE.MeshStandardMaterial({ color: 0x333333 });
        const base = new THREE.Mesh(baseGeo, baseMat);
        base.position.set(0, -0.6, 0);
        mesh.add(base);

        // Label flutuante 
        this.createLabel(mesh, name);

        //Indicador de status (led) 
        const ledGeo = new THREE.SphereGeometry(0.2, 8, 8);
        const ledMat = new THREE.MeshStandardMaterial({ color: 0x00ff00 });
        const led = new THREE.Mesh(ledGeo, ledMat);
        led.position.y = 1.5;
        led.name = "statusLed";
        mesh.add(led);


        this.scene.add(mesh);
        return mesh;
    }

    createLabel(parent, text) {
        const canvas = document.createElement('canvas');
        const conext = canvas.getContext('2d');
        canvas.width = 256;
        canvas.height = 64;

        context.fillStyle = 'rgba(0,0,0,0.7)';
        context.fillRect(0, 0, canvas.width, canvas.height);
        context.fillStyle = '#fff';
        context.font = '24px Arial';
        context.textAlign = 'center';
        context.fillText(text, 128, 40);

        const texture = new THREE.CanvasTexture(canvas);
        const spriteMaterial = new THREE.SpriteMaterial({ map: texture });
        const sprite = new THREE.Sprite(spriteMaterial);
        sprite.scale.set(4, 1, 1);
        sprite.position.y = 3;


        parent.add(sprite);
        // sprite é  um tipo especial de objeto 3D que sempre enfrenta a câmera, ideal para rótulos e ícones. Ele é criado usando uma textura gerada a partir de um canvas HTML, onde o texto do rótulo é desenhado. O sprite é então posicionado acima do sensor para servir como um rótulo flutuante que identifica o tipo do sensor.
    }

    setupInteraction() {
        const canvas = document.getElementsById('canvas');

        canvas.addEventListener('mousedown', (e) => this.onMouseDown(e));
        canvas.addEventListener('mousemove', (e) => this.onMouseMove(e));
        canvas.addEventListener('mouseup', (e) => this.onMouseUp(e));

        // raycasting para grag and drop 
        window.addEventListener('mousemove', (e) => {
            this.mouse.x = (e.clientX / window.innerWidth) * 2 - 1;
            this.mouse.y = -(e.clientY / window.innerHeight) * 2 + 1;
        });
    }

    onMouseDown(event) {
        this.raycaster.setFromCamera(this.mouse, this.camera);
        const intersects = this.raycaster.intersectObjects(this.scene.children, true);

        for (let intersect of intersects) {
            // Encontra o mesh do sensor (pode estar em grupo)
            let obj = intersect.object;
            while (obj && !obj.userData.isSensor) {
                obj = obj.parent;
            }

            if (obj && obj.userData.isSensor) {
                this.selectedSensor = obj;
                this.isDragging = true;
                document.body.style.cursor = 'move';
                break;
            }
        }
    }

    onMouseMove(event) {
        if (!this.isDragging || !this.selectedSensor) return;

        this.raycaster.setFromCamera(this.mouse, this.camera);
        const intersects = this.raycaster.intersectObject(this.terrain);

        if (intersects.length > 0) {
            const point = intersects[0].point;
            // Mantém a altura relativa do sensor
            const currentY = this.selectedSensor.position.y;
            const terrainY = intersects[0].point.y;

            this.selectedSensor.position.set(point.x, terrainY + 1.5, point.z);

            // Envia atualização para backend
            this.updateSensorPosition(this.selectedSensor.userData.id, {
                x: point.x,
                y: terrainY + 1.5,
                z: point.z
            });
        }
    }

    onMouseUp(event) {
        if (this.isDragging && this.selectedSensor) {
            this.isDragging = false;
            this.selectedSensor = null;
            document.body.style.cursor = 'default';
        }
    }

    updateSensorStatus(sensorId, active, reading) {
        const sensor = this.findSensorById(sensorId);
        if (!sensor) return;

        const led = sensor.getObjectByName('statusLed');
        if (led) {
            led.material.color.setHex(active ? 0x00ff00 : 0x333333);
        }

        // Animação de pulso se ativo
        if (active) {
            this.pulseAnimation(sensor);
        }
    }

    pulseAnimation(sensor) {
        // Implementação de animação de pulso quando recebe dados
    }

    updateSensorPosition(id, pos) {
        // Chama API REST para atualizar posição
        fetch(`/api/sensors/${id}/move`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(pos)
        });
    }

    findSensorById(id) {
        return this.scene.children.find(
            child => child.userData.id === id
        );
    }
}
