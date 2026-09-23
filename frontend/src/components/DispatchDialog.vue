<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { ElMessage } from 'element-plus';
import type { FormInstance, FormRules } from 'element-plus';
import { confirmTask, reassignTask } from '../services/storage.service';
import { DISPATCH_ACTION_REASSIGN } from '../constants/app.constants';
import type { DispatchRequest, Driver, FarmTask, Machine } from '../types/domain';

const props = defineProps<{
  visible: boolean;
  task: FarmTask | null;
  machines: Machine[];
  drivers: Driver[];
}>();

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void;
  (e: 'dispatched'): void;
}>();

const formRef = ref<FormInstance>();
const submitting = ref(false);

const form = reactive<DispatchRequest>({
  machineCode: '',
  driverName: '',
  reason: '',
});

const isReassign = () => props.task?.status === '已派单';

const rules: FormRules = {
  reason: [{ required: true, message: '改派必须填写改派原因', trigger: 'blur' }],
};

// 弹窗打开时按任务回填默认值（未传资源时后端沿用推荐资源）。
watch(
  () => props.visible,
  (visible) => {
    if (visible && props.task) {
      form.machineCode = '';
      form.driverName = '';
      form.reason = '';
      formRef.value?.clearValidate();
    }
  },
);

const idleMachines = () => props.machines.filter((m) => m.status === '空闲');
const availableDrivers = () => props.drivers.filter((d) => d.status === '在岗' || d.status === '可派单');

const dialogTitle = computed(() => {
  if (!props.task) return '';
  const action = isReassign() ? '任务改派' : '派单确认';
  return `${action} · ${props.task.type} ${props.task.field}`;
});

const close = () => emit('update:visible', false);

const handleSubmit = async () => {
  if (!props.task) return;
  if (isReassign()) {
    const valid = await formRef.value?.validate().catch(() => false);
    if (!valid) return;
  }
  submitting.value = true;
  try {
    const payload: DispatchRequest = {
      machineCode: form.machineCode || undefined,
      driverName: form.driverName || undefined,
      reason: form.reason || undefined,
    };
    const result = isReassign()
      ? await reassignTask(props.task.id, payload)
      : await confirmTask(props.task.id, payload);
    ElMessage.success(result.message);
    close();
    emit('dispatched');
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '操作失败');
  } finally {
    submitting.value = false;
  }
};
</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="dialogTitle"
    width="460px"
    @close="close"
  >
    <el-form
      v-if="task"
      ref="formRef"
      :model="form"
      :rules="isReassign() ? rules : {}"
      label-width="84px"
      @submit.prevent
    >
      <el-form-item label="任务地块">
        <span class="text-sm text-slate-600">{{ task.type }} · {{ task.field }} · {{ task.areaMu }} 亩</span>
      </el-form-item>
      <el-form-item label="推荐农机">
        <span class="text-sm text-emerald-700">{{ task.recommendedMachine || '无' }}</span>
      </el-form-item>
      <el-form-item label="推荐驾驶员">
        <span class="text-sm text-emerald-700">{{ task.recommendedDriver || '无' }}</span>
      </el-form-item>
      <el-form-item label="指定农机">
        <el-select v-model="form.machineCode" clearable placeholder="不选则沿用推荐农机" class="w-full">
          <el-option
            v-for="m in idleMachines()"
            :key="m.code"
            :label="`${m.code} ${m.name}`"
            :value="m.code"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="指定驾驶员">
        <el-select v-model="form.driverName" clearable placeholder="不选则沿用推荐驾驶员" class="w-full">
          <el-option v-for="d in availableDrivers()" :key="d.id" :label="`${d.name}（${d.status}）`" :value="d.name" />
        </el-select>
      </el-form-item>
      <el-form-item v-if="isReassign()" label="当前资源">
        <span class="text-sm text-amber-700">{{ task.assignedMachine }} / {{ task.assignedDriver }}</span>
      </el-form-item>
      <el-form-item :label="isReassign() ? '改派原因' : '派单备注'">
        <el-input
          v-model="form.reason"
          type="textarea"
          :rows="3"
          maxlength="200"
          show-word-limit
          :placeholder="isReassign() ? '请填写改派原因（必填）' : '可填写派单说明，不填使用默认原因'"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">
        {{ isReassign() ? DISPATCH_ACTION_REASSIGN : '确认派单' }}
      </el-button>
    </template>
  </el-dialog>
</template>
