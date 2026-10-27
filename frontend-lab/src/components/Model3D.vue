<script setup>
// Visualizador 3D (STL, OBJ, GLB/GLTF, PLY): arrastar gira, rolar dá zoom.
import { ref, watch, onMounted, onBeforeUnmount } from "vue";
const props = defineProps({ url: String, name: String });
const host = ref(null);
const state = ref("loading");
let T = null, renderer, scene, camera, controls, raf = 0, obj = null, ro = null;

async function loadModel() {
  state.value = "loading";
  try {
    const ext = String(props.name).toLowerCase().split(".").pop();
    const [three, { OrbitControls }] = await Promise.all([import("three"), import("three/examples/jsm/controls/OrbitControls.js")]);
    T = three;
    if (!renderer) {
      renderer = new T.WebGLRenderer({ antialias: true, alpha: true });
      renderer.setPixelRatio(Math.min(2, window.devicePixelRatio || 1));
      host.value.appendChild(renderer.domElement);
      scene = new T.Scene();
      camera = new T.PerspectiveCamera(45, 1, 0.01, 10000);
      scene.add(new T.HemisphereLight(0xffffff, 0x444466, 1.1));
      const d = new T.DirectionalLight(0xffffff, 1.4); d.position.set(3, 5, 4); scene.add(d);
      controls = new OrbitControls(camera, renderer.domElement); controls.enableDamping = true;
      ro = new ResizeObserver(resize); ro.observe(host.value); resize();
      const tick = () => { raf = requestAnimationFrame(tick); controls.update(); renderer.render(scene, camera); };
      tick();
    }
    const res = await fetch(props.url, { credentials: "same-origin" });
    if (!res.ok) throw new Error(String(res.status));
    const buf = await res.arrayBuffer();
    const mat = new T.MeshStandardMaterial({ color: 0x9a86ff, roughness: 0.55, metalness: 0.1 });
    let o;
    if (ext === "stl") { const { STLLoader } = await import("three/examples/jsm/loaders/STLLoader.js"); const g = new STLLoader().parse(buf); g.computeVertexNormals(); o = new T.Mesh(g, mat); }
    else if (ext === "ply") { const { PLYLoader } = await import("three/examples/jsm/loaders/PLYLoader.js"); const g = new PLYLoader().parse(buf); g.computeVertexNormals(); o = new T.Mesh(g, mat); }
    else if (ext === "obj") { const { OBJLoader } = await import("three/examples/jsm/loaders/OBJLoader.js"); o = new OBJLoader().parse(new TextDecoder().decode(buf)); o.traverse((c) => { if (c.isMesh) c.material = mat; }); }
    else { const { GLTFLoader } = await import("three/examples/jsm/loaders/GLTFLoader.js"); o = (await new GLTFLoader().parseAsync(buf, "")).scene; }
    if (obj) scene.remove(obj);
    obj = o; scene.add(o);
    const box = new T.Box3().setFromObject(o); const size = box.getSize(new T.Vector3()); const ctr = box.getCenter(new T.Vector3());
    o.position.sub(ctr);
    const r = Math.max(size.x, size.y, size.z) || 1;
    camera.position.set(r * 1.3, r * 0.9, r * 1.6); camera.near = r / 100; camera.far = r * 100; camera.updateProjectionMatrix();
    controls.target.set(0, 0, 0); controls.update();
    state.value = "ok";
  } catch (e) { state.value = "err"; }
}
function resize() {
  if (!renderer || !host.value) return;
  const w = host.value.clientWidth, h = host.value.clientHeight;
  renderer.setSize(w, h); camera.aspect = w / Math.max(1, h); camera.updateProjectionMatrix();
}
onMounted(loadModel);
watch(() => props.url, loadModel);
onBeforeUnmount(() => { cancelAnimationFrame(raf); if (ro) ro.disconnect(); if (renderer) { renderer.dispose(); renderer.domElement.remove(); } });
</script>

<template>
  <div class="m3d">
    <div ref="host" class="m3d-host"></div>
    <div v-if="state === 'loading'" class="vw-msg m3d-msg">Carregando modelo…</div>
    <div v-else-if="state === 'err'" class="vw-msg m3d-msg">Não foi possível mostrar este modelo 3D.</div>
  </div>
</template>
