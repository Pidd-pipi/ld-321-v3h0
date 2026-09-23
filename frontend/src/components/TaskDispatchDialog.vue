<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import type { FormInstance, FormRules } from 'element-plus';
import type { DispatchResult, Driver, FarmTask, Machine } from '../types/domain';
import { dispatchTask, reassignTask } from '../services/storage.service';

const props = defineProps<{
  visible: boolean;
  mode: 'dispatch' | 'reassign';
  task: FarmTask | null;
  machines: Machine[];
  drivers: Driver[];
}>();

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void;
  (e: 'success', result: DispatchResult): void;
  (e: 'error', err: unknown): void;
}>();

const formRef = ref<FormInstance>();
const submitting = ref(false);

const form = reactive({
  machineCode: '',
  driverName: '',
  reason: '',
});

const title = computed(() => (props.mode === 'dispatch' ? '确认派单' : '任务改派'));

// 派单与改派的目标资源都必须空闲/可派单；后端事务会再次强校验。
const idleMachines = computed(() => props.machines.filter((m) => m.status === '空闲'));

const availableDrivers = computed(() =>
  props.drivers.filter((d) => d.status === '在岗' || d.status === '可派单'),
);

const rules = computed<FormRules>(() =>
  props.mode === 'reassign'
    ? { reason: [{ required: true, message: '改派原因不能为空', trigger: 'blur' }] }
    : {},
);

// 每次打开时重置表单（不选即沿用推荐/当前资源）。
watch(
  () => props.visible,
  (open) => {
    if (!open) return;
    form.machineCode = '';
    form.driverName = '';
    form.reason = '';
    formRef.value?.clearValidate();
  },
);

const close = () => emit('update:visible', false);

const handleSubmit = async () => {
  if (!props.task) return;
  const valid = await formRef.value?.validate().catch(() => false);
  if (!valid) return;
  submitting.value = true;
  try {
    const payload = {
      machineCode: form.machineCode || undefined,
      driverName: form.driverName || undefined,
      reason: form.reason || undefined,
    };
    const result =
      props.mode === 'dispatch'
        ? await dispatchTask(props.task.id, payload)
        : await reassignTask(props.task.id, { ...payload, reason: form.reason });
    emit('success', result);
    close();
  } catch (err) {
    // 失败不关闭弹窗、不改本地状态，仅把明确错误上抛给卡片提示。
    emit('error', err);
  } finally {
    submitting.value = false;
  }
};
</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="title"
    width="460px"
    @update:model-value="emit('update:visible', $event)"
  >
    <el-form v-if="task" ref="formRef" :model="form" :rules="rules" label-width="88px" @submit.prevent>
      <el-form-item label="任务">
        <span class="font-bold">{{ task.type }} · {{ task.field }}</span>
      </el-form-item>
      <el-form-item label="农机">
        <el-select
          v-model="form.machineCode"
          :placeholder="mode === 'reassign' ? '不选则沿用当前农机' : '不选则沿用推荐农机'"
          clearable
          class="w-full"
        >
          <el-option
            v-for="m in idleMachines"
            :key="m.code"
            :label="`${m.code} ${m.name}（${m.status}）`"
            :value="m.code"
          />
        </el-select>
        <p class="mt-1 w-full text-xs text-slate-400">
          {{ mode === 'dispatch' ? `推荐：${task.recommendedMachine || '—'}` : `当前：${task.assignedMachine || task.recommendedMachine || '—'}` }}
        </p>
      </el-form-item>
      <el-form-item label="驾驶员">
        <el-select
          v-model="form.driverName"
          :placeholder="mode === 'reassign' ? '不选则沿用当前驾驶员' : '不选则沿用推荐驾驶员'"
          clearable
          class="w-full"
        >
          <el-option
            v-for="d in availableDrivers"
            :key="d.id"
            :label="`${d.name}（${d.status}）`"
            :value="d.name"
          />
        </el-select>
        <p class="mt-1 w-full text-xs text-slate-400">
          {{ mode === 'dispatch' ? `推荐：${task.recommendedDriver || '—'}` : `当前：${task.assignedDriver || task.recommendedDriver || '—'}` }}
        </p>
      </el-form-item>
      <el-form-item label="原因" prop="reason">
        <el-input
          v-model="form.reason"
          type="textarea"
          :rows="3"
          maxlength="255"
          show-word-limit
          :placeholder="mode === 'reassign' ? '请填写改派原因（必填）' : '可填写派单说明，选填'"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">
        {{ mode === 'dispatch' ? '确认派单' : '确认改派' }}
      </el-button>
    </template>
  </el-dialog>
</template>
