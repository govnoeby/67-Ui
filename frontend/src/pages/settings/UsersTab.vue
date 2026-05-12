<script setup>
import { onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { Modal, message } from 'ant-design-vue';
import { HttpUtil } from '@/utils';

const { t } = useI18n();

const users = ref([]);
const loading = ref(false);
const modalOpen = ref(false);
const editingUser = ref(null);

const form = ref({
  username: '',
  password: '',
  email: '',
  role: 'operator',
});

async function fetchUsers() {
  loading.value = true;
  try {
    const msg = await HttpUtil.get('/panel/setting/users');
    if (msg?.success) users.value = msg.obj || [];
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  editingUser.value = null;
  form.value = { username: '', password: '', email: '', role: 'operator' };
  modalOpen.value = true;
}

function openEdit(user) {
  editingUser.value = user;
  form.value = { username: user.username, password: '', email: user.email || '', role: user.role };
  modalOpen.value = true;
}

async function saveUser() {
  if (editingUser.value) {
    const payload = { ...form.value };
    if (!payload.password) delete payload.password;
    const msg = await HttpUtil.put(`/panel/setting/users/${editingUser.value.id}`, payload);
    if (msg?.success) {
      message.success('User updated');
      modalOpen.value = false;
      await fetchUsers();
    } else {
      message.error(msg?.msg || 'Failed to update user');
    }
  } else {
    const msg = await HttpUtil.post('/panel/setting/users', form.value);
    if (msg?.success) {
      message.success('User created');
      modalOpen.value = false;
      await fetchUsers();
    } else {
      message.error(msg?.msg || 'Failed to create user');
    }
  }
}

function confirmDelete(user) {
  Modal.confirm({
    title: t('delete'),
    content: `Delete user "${user.username}"?`,
    okType: 'danger',
    async onOk() {
      const msg = await HttpUtil.del(`/panel/setting/users/${user.id}`);
      if (msg?.success) {
        message.success('User deleted');
        await fetchUsers();
      } else {
        message.error(msg?.msg || 'Failed to delete user');
      }
    },
  });
}

async function toggleActive(user) {
  const msg = await HttpUtil.put(`/panel/setting/users/${user.id}`, { isActive: !user.isActive });
  if (msg?.success) {
    user.isActive = !user.isActive;
  }
}

onMounted(fetchUsers);
</script>

<template>
  <div class="users-tab">
    <a-button type="primary" @click="openCreate" style="margin-bottom: 16px">
      Create User
    </a-button>

    <a-table :data-source="users" :loading="loading" row-key="id" :pagination="{ pageSize: 25 }">
      <a-table-column title="ID" data-index="id" :width="60" />
      <a-table-column title="Username" data-index="username" />
      <a-table-column title="Email" data-index="email" />
      <a-table-column title="Role" data-index="role">
        <template #default="{ text }">
          <a-tag :color="text === 'admin' ? 'red' : text === 'operator' ? 'blue' : 'green'">
            {{ text }}
          </a-tag>
        </template>
      </a-table-column>
      <a-table-column title="Active" data-index="isActive" :width="80">
        <template #default="{ record }">
          <a-switch :checked="record.isActive" @change="() => toggleActive(record)" />
        </template>
      </a-table-column>
      <a-table-column title="Created" data-index="createdAt" :width="160">
        <template #default="{ text }">{{ text ? new Date(text).toLocaleString() : '-' }}</template>
      </a-table-column>
      <a-table-column title="Actions" :width="160">
        <template #default="{ record }">
          <a-space>
            <a-button size="small" @click="openEdit(record)">Edit</a-button>
            <a-button size="small" danger :disabled="record.role === 'admin'"
              @click="confirmDelete(record)">Delete</a-button>
          </a-space>
        </template>
      </a-table-column>
    </a-table>

    <a-modal v-model:open="modalOpen" :title="editingUser ? 'Edit User' : 'Create User'" @ok="saveUser">
      <a-form layout="vertical">
        <a-form-item label="Username" required>
          <a-input v-model:value="form.username" />
        </a-form-item>
        <a-form-item label="Password" :required="!editingUser">
          <a-input-password v-model:value="form.password" :placeholder="editingUser ? 'Leave empty to keep current' : ''" />
        </a-form-item>
        <a-form-item label="Email">
          <a-input v-model:value="form.email" />
        </a-form-item>
        <a-form-item label="Role">
          <a-select v-model:value="form.role">
            <a-select-option value="admin">Admin</a-select-option>
            <a-select-option value="operator">Operator</a-select-option>
            <a-select-option value="viewer">Viewer</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
