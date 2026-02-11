const taskForm = document.getElementById("task-form");
const taskInput = document.getElementById("task-input");
const taskList = document.getElementById("task-list");
const template = document.getElementById("task-item-template");

async function fetchTasks() {
  const res = await fetch("/api/tasks");
  if (!res.ok) {
    throw new Error("Failed to load tasks");
  }
  const data = await res.json();
  return data.tasks || [];
}

async function createTask(title) {
  const res = await fetch("/api/tasks", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title }),
  });

  if (!res.ok) {
    const errText = await res.text();
    throw new Error(errText || "Failed to create task");
  }

  return res.json();
}

async function toggleTask(id) {
  const res = await fetch(`/api/tasks/${id}`, { method: "PATCH" });
  if (!res.ok) {
    throw new Error("Failed to update task");
  }
}

async function deleteTask(id) {
  const res = await fetch(`/api/tasks/${id}`, { method: "DELETE" });
  if (!res.ok) {
    throw new Error("Failed to delete task");
  }
}

function renderTasks(tasks) {
  taskList.innerHTML = "";

  if (!tasks.length) {
    const empty = document.createElement("li");
    empty.textContent = "No tasks yet.";
    empty.style.opacity = "0.8";
    empty.style.padding = "0.4rem";
    taskList.appendChild(empty);
    return;
  }

  tasks.forEach((task) => {
    const node = template.content.firstElementChild.cloneNode(true);
    const checkbox = node.querySelector(".task-toggle");
    const title = node.querySelector(".task-title");
    const deleteBtn = node.querySelector(".delete-btn");

    checkbox.checked = task.done;
    title.textContent = task.title;
    title.classList.toggle("done", task.done);

    checkbox.addEventListener("change", async () => {
      try {
        await toggleTask(task.id);
        await refresh();
      } catch (err) {
        alert(err.message);
      }
    });

    deleteBtn.addEventListener("click", async () => {
      try {
        await deleteTask(task.id);
        await refresh();
      } catch (err) {
        alert(err.message);
      }
    });

    taskList.appendChild(node);
  });
}

async function refresh() {
  try {
    const tasks = await fetchTasks();
    renderTasks(tasks);
  } catch (err) {
    taskList.innerHTML = `<li style="color:#fecaca">${err.message}</li>`;
  }
}

taskForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  const title = taskInput.value.trim();
  if (!title) {
    return;
  }

  try {
    await createTask(title);
    taskInput.value = "";
    await refresh();
  } catch (err) {
    alert(err.message);
  }
});

refresh();
