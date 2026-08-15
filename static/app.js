document.addEventListener('DOMContentLoaded', () => {
  // Elementos do DOM
  const loginModal = document.getElementById('login-modal');
  const loginForm = document.getElementById('login-form');
  const usernameInput = document.getElementById('username-input');
  const avatarPreview = document.getElementById('avatar-preview');

  const appContainer = document.getElementById('app-container');
  const sidebar = document.getElementById('sidebar');
  const toggleSidebarBtn = document.getElementById('toggle-sidebar-btn');

  const myAvatar = document.getElementById('my-avatar');
  const myUsername = document.getElementById('my-username');

  const userList = document.getElementById('user-list');
  const searchUsers = document.getElementById('search-users');
  const onlineCounterBadge = document.getElementById('online-counter-badge');
  const chatSubtitle = document.getElementById('chat-subtitle');

  const messagesContainer = document.getElementById('messages-container');
  const messageForm = document.getElementById('message-form');
  const messageInput = document.getElementById('message-input');
  const logoutBtn = document.getElementById('logout-btn');
  const clearHistoryBtn = document.getElementById('clear-history-btn');

  let ws = null;
  let currentUsername = '';
  let activeUsers = [];

  // Cores dinâmicas para avatares baseadas na inicial
  const colors = [
    '#10B981', '#3B82F6', '#8B5CF6', '#EC4899',
    '#F59E0B', '#06B6D4', '#6366F1', '#F43F5E',
    '#14B8A6', '#84CC16', '#D97706', '#A855F7'
  ];

  function getAvatarColor(str) {
    if (!str) return colors[0];
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      hash = str.charCodeAt(i) + ((hash << 5) - hash);
    }
    const index = Math.abs(hash) % colors.length;
    return colors[index];
  }

  // Atualiza pré-visualização do avatar no login ao digitar
  usernameInput.addEventListener('input', (e) => {
    const value = e.target.value.trim();
    if (value.length > 0) {
      const initial = value.charAt(0).toUpperCase();
      avatarPreview.textContent = initial;
      avatarPreview.style.backgroundColor = getAvatarColor(value);
    } else {
      avatarPreview.textContent = '?';
      avatarPreview.style.backgroundColor = '#00a884';
    }
  });

  // Submissão do Formulário de Login
  loginForm.addEventListener('submit', (e) => {
    e.preventDefault();
    const name = usernameInput.value.trim();
    if (!name) return;

    currentUsername = name;
    connectWebSocket(name);
  });

  // Conectar ao WebSocket
  function connectWebSocket(username) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws?username=${encodeURIComponent(username)}`;

    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      // Sucesso na conexão
      loginModal.classList.add('hidden');
      appContainer.classList.remove('hidden');

      const initial = username.charAt(0).toUpperCase();
      const userColor = getAvatarColor(username);

      myAvatar.textContent = initial;
      myAvatar.style.backgroundColor = userColor;
      myUsername.textContent = username;

      messageInput.focus();
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        handleIncomingMessage(data);
      } catch (err) {
        console.error('Erro ao processar mensagem do servidor:', err);
      }
    };

    ws.onclose = () => {
      appendSystemMessage('Conexão perdida com o servidor.');
      chatSubtitle.textContent = 'Desconectado';
    };

    ws.onerror = (error) => {
      console.error('Erro no WebSocket:', error);
    };
  }

  // Processa mensagens recebidas do servidor
  function handleIncomingMessage(msg) {
    switch (msg.type) {
      case 'history':
        if (msg.history && Array.isArray(msg.history)) {
          msg.history.forEach(item => handleIncomingMessage(item));
        }
        break;

      case 'message':
        renderChatMessage(msg);
        break;

      case 'join':
        if (msg.sender !== currentUsername) {
          appendSystemMessage(`${msg.sender} entrou na sala`);
        }
        break;

      case 'leave':
        if (msg.sender !== currentUsername) {
          appendSystemMessage(`${msg.sender} saiu da sala`);
        }
        break;

      case 'history_cleared':
        messagesContainer.innerHTML = `
          <div class="system-date-divider">
            <span>HOJE</span>
          </div>
        `;
        appendSystemMessage(`${msg.sender || 'Um participante'} apagou todo o histórico da conversa`);
        break;

      case 'user_list':
        updateUserList(msg.users || []);
        break;

      default:
        console.log('Tipo de mensagem desconhecido:', msg);
    }
  }

  // Renderiza um balão de conversa de mensagem
  function renderChatMessage(msg) {
    const isMe = msg.sender === currentUsername;
    const wrapper = document.createElement('div');
    wrapper.className = `message-bubble-wrapper ${isMe ? 'sent' : 'received'}`;

    const bubble = document.createElement('div');
    bubble.className = 'message-bubble';

    // Nome do remetente (apenas para mensagens recebidas)
    if (!isMe) {
      const senderDiv = document.createElement('div');
      senderDiv.className = 'sender-name';
      senderDiv.style.color = msg.color || getAvatarColor(msg.sender);
      senderDiv.textContent = msg.sender;
      bubble.appendChild(senderDiv);
    }

    // Conteúdo da mensagem
    const textNode = document.createElement('span');
    textNode.textContent = msg.content;
    bubble.appendChild(textNode);

    // Meta-informações (Hora e Checkmarks se for mensagem enviada)
    const metaDiv = document.createElement('div');
    metaDiv.className = 'message-meta';
    
    const timeSpan = document.createElement('span');
    timeSpan.textContent = msg.timestamp || new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    metaDiv.appendChild(timeSpan);

    if (isMe) {
      const checkSpan = document.createElement('span');
      checkSpan.className = 'check-marks';
      checkSpan.innerHTML = `
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <polyline points="20 6 9 17 4 12"></polyline>
          <polyline points="20 12 13.5 18.5"></polyline>
        </svg>
      `;
      metaDiv.appendChild(checkSpan);
    }

    bubble.appendChild(metaDiv);
    wrapper.appendChild(bubble);
    messagesContainer.appendChild(wrapper);

    scrollToBottom();
  }

  // Adiciona notificação de sistema no centro do chat
  function appendSystemMessage(text) {
    const badge = document.createElement('div');
    badge.className = 'system-badge';
    badge.textContent = text;
    messagesContainer.appendChild(badge);
    scrollToBottom();
  }

  // Atualiza lista de utilizadores na sidebar
  function updateUserList(users) {
    activeUsers = users;
    renderUserListFiltered();

    const count = users.length;
    onlineCounterBadge.textContent = count;
    chatSubtitle.textContent = `${count} ${count === 1 ? 'participante online' : 'participantes online'}`;
  }

  // Renderiza a lista de utilizadores com filtro de pesquisa
  function renderUserListFiltered() {
    const query = searchUsers.value.toLowerCase().trim();
    userList.innerHTML = '';

    const filtered = activeUsers.filter(u => u.toLowerCase().includes(query));

    filtered.forEach(name => {
      const li = document.createElement('li');
      li.className = 'user-item';

      const initial = name.charAt(0).toUpperCase();
      const color = getAvatarColor(name);

      const avatar = document.createElement('div');
      avatar.className = 'user-avatar';
      avatar.style.backgroundColor = color;
      avatar.textContent = initial;

      const nameSpan = document.createElement('span');
      nameSpan.className = 'user-name';
      nameSpan.textContent = name === currentUsername ? `${name} (Você)` : name;

      li.appendChild(avatar);
      li.appendChild(nameSpan);
      userList.appendChild(li);
    });
  }

  searchUsers.addEventListener('input', renderUserListFiltered);

  // Envio de mensagens
  messageForm.addEventListener('submit', (e) => {
    e.preventDefault();
    const content = messageInput.value.trim();
    if (!content || !ws || ws.readyState !== WebSocket.OPEN) return;

    const msg = {
      content: content
    };

    ws.send(JSON.stringify(msg));
    messageInput.value = '';
    messageInput.focus();
  });

  // Scroll automático para a última mensagem
  function scrollToBottom() {
    messagesContainer.scrollTop = messagesContainer.scrollHeight;
  }

  // Botão de Logout / Sair da Sala
  logoutBtn.addEventListener('click', () => {
    if (ws) {
      ws.close();
    }
    appContainer.classList.add('hidden');
    loginModal.classList.remove('hidden');
    messagesContainer.innerHTML = `
      <div class="system-date-divider">
        <span>HOJE</span>
      </div>
    `;
    usernameInput.value = '';
    avatarPreview.textContent = '?';
    avatarPreview.style.backgroundColor = '#00a884';
  });

  // Botão de Apagar Histórico da Sala
  clearHistoryBtn.addEventListener('click', () => {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    const confirmClear = confirm('Tem a certeza que deseja apagar todo o histórico de mensagens para todos no grupo?');
    if (confirmClear) {
      ws.send(JSON.stringify({ type: 'clear_history' }));
    }
  });

  // Mobile menu toggle
  toggleSidebarBtn.addEventListener('click', () => {
    sidebar.classList.toggle('active');
  });

  // Fechar sidebar ao clicar fora em telas pequenas
  messagesContainer.addEventListener('click', () => {
    if (window.innerWidth <= 768 && sidebar.classList.contains('active')) {
      sidebar.classList.remove('active');
    }
  });
});
