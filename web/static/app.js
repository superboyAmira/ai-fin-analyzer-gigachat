// API Base URL - use current origin for production
const API_BASE = window.location.origin;

// Token management
function getToken() {
    return localStorage.getItem('accessToken');
}

function setToken(token) {
    localStorage.setItem('accessToken', token);
}

function getRefreshToken() {
    return localStorage.getItem('refreshToken');
}

function setRefreshToken(token) {
    localStorage.setItem('refreshToken', token);
}

function clearTokens() {
    localStorage.removeItem('accessToken');
    localStorage.removeItem('refreshToken');
    localStorage.removeItem('userEmail');
}

// Check authentication on page load
function checkAuth() {
    const token = getToken();
    if (token) {
        showApp();
        loadDocuments();
    } else {
        showLogin();
    }
}

// Show/hide sections
function showLogin() {
    document.getElementById('loginSection').style.display = 'block';
    document.getElementById('registerSection').style.display = 'none';
    document.getElementById('appSection').style.display = 'none';
    document.getElementById('userInfo').style.display = 'none';
}

function showRegister() {
    document.getElementById('loginSection').style.display = 'none';
    document.getElementById('registerSection').style.display = 'block';
    document.getElementById('appSection').style.display = 'none';
    document.getElementById('userInfo').style.display = 'none';
}

function showApp() {
    document.getElementById('loginSection').style.display = 'none';
    document.getElementById('registerSection').style.display = 'none';
    document.getElementById('appSection').style.display = 'block';
    document.getElementById('userInfo').style.display = 'flex';
    
    const email = localStorage.getItem('userEmail');
    if (email) {
        document.getElementById('userEmail').textContent = email;
    }
}

// API calls with token refresh
async function apiCall(url, options = {}) {
    const token = getToken();
    
    const headers = {
        'Content-Type': 'application/json',
        ...options.headers,
    };
    
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }
    
    const response = await fetch(`${API_BASE}${url}`, {
        ...options,
        headers,
    });
    
    // If unauthorized, try to refresh token
    if (response.status === 401 && token) {
        const refreshed = await refreshAccessToken();
        if (refreshed) {
            // Retry with new token
            headers['Authorization'] = `Bearer ${getToken()}`;
            return fetch(`${API_BASE}${url}`, {
                ...options,
                headers,
            });
        } else {
            // Refresh failed, logout
            logout();
            throw new Error('Session expired');
        }
    }
    
    return response;
}

async function refreshAccessToken() {
    const refreshToken = getRefreshToken();
    if (!refreshToken) {
        return false;
    }
    
    try {
        const response = await fetch(`${API_BASE}/user/auth/refresh`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ refresh_token: refreshToken }),
        });
        
        if (response.ok) {
            const data = await response.json();
            setToken(data.access_token);
            setRefreshToken(data.refresh_token);
            return true;
        }
    } catch (error) {
        console.error('Token refresh failed:', error);
    }
    
    return false;
}

// Login handler
async function handleLogin(event) {
    event.preventDefault();
    const errorDiv = document.getElementById('loginError');
    errorDiv.style.display = 'none';
    
    const email = document.getElementById('email').value;
    const password = document.getElementById('password').value;
    
    try {
        const response = await fetch(`${API_BASE}/user/auth/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ email, password }),
        });
        
        const data = await response.json();
        
        if (response.ok) {
            setToken(data.access_token);
            setRefreshToken(data.refresh_token);
            localStorage.setItem('userEmail', data.user.email);
            showApp();
            loadDocuments();
        } else {
            errorDiv.textContent = data.error || 'Ошибка входа';
            errorDiv.style.display = 'block';
        }
    } catch (error) {
        errorDiv.textContent = 'Ошибка подключения к серверу';
        errorDiv.style.display = 'block';
    }
}

// Register handler
async function handleRegister(event) {
    event.preventDefault();
    const errorDiv = document.getElementById('registerError');
    errorDiv.style.display = 'none';
    
    const username = document.getElementById('regUsername').value;
    const email = document.getElementById('regEmail').value;
    const password = document.getElementById('regPassword').value;
    
    try {
        const response = await fetch(`${API_BASE}/user/auth/register`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ username, email, password }),
        });
        
        const data = await response.json();
        
        if (response.ok) {
            setToken(data.access_token);
            setRefreshToken(data.refresh_token);
            localStorage.setItem('userEmail', data.user.email);
            showApp();
            loadDocuments();
        } else {
            errorDiv.textContent = data.error || 'Ошибка регистрации';
            errorDiv.style.display = 'block';
        }
    } catch (error) {
        errorDiv.textContent = 'Ошибка подключения к серверу';
        errorDiv.style.display = 'block';
    }
}

// Logout handler
function logout() {
    clearTokens();
    showLogin();
    document.getElementById('email').value = 'nikita2@gmail.com';
    document.getElementById('password').value = '128912';
}

// Upload handler
async function handleUpload(event) {
    event.preventDefault();
    const errorDiv = document.getElementById('uploadError');
    const successDiv = document.getElementById('uploadSuccess');
    errorDiv.style.display = 'none';
    successDiv.style.display = 'none';
    
    const fileInput = document.getElementById('file');
    const docType = document.getElementById('docType').value;
    
    if (!fileInput.files[0]) {
        errorDiv.textContent = 'Выберите файл';
        errorDiv.style.display = 'block';
        return;
    }
    
    const formData = new FormData();
    formData.append('file', fileInput.files[0]);
    formData.append('type', docType);
    
    try {
        const token = getToken();
        const response = await fetch(`${API_BASE}/api/v1/documents/upload`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`,
            },
            body: formData,
        });
        
        const data = await response.json();
        
        if (response.ok) {
            successDiv.textContent = 'Документ успешно загружен! Обработка началась...';
            successDiv.style.display = 'block';
            fileInput.value = '';
            
            // Reload documents list first
            await loadDocuments();
            
            // Auto-process document
            setTimeout(() => {
                processDocument(data.id, null);
            }, 500);
        } else {
            errorDiv.textContent = data.error || 'Ошибка загрузки';
            errorDiv.style.display = 'block';
        }
    } catch (error) {
        errorDiv.textContent = 'Ошибка подключения к серверу';
        errorDiv.style.display = 'block';
    }
}

// Store processed document data
let processedDocuments = {};

// Process document
async function processDocument(documentId, event) {
    // Show loading state
    const button = event?.target || document.querySelector(`button[onclick*="processDocument('${documentId}')"]`);
    if (button) {
        button.disabled = true;
        button.innerHTML = '<span class="spinner"></span> Обработка...';
    }
    
    // Show loading indicator in document card
    const docCard = document.querySelector(`[data-doc-id="${documentId}"]`);
    if (docCard) {
        const statusEl = docCard.querySelector('.status');
        if (statusEl) {
            statusEl.textContent = 'Обрабатывается...';
            statusEl.className = 'status processing';
        }
    }
    
    try {
        const response = await apiCall(`/api/v1/documents/${documentId}/process`, {
            method: 'POST',
        });
        
        if (response.ok) {
            const data = await response.json();
            console.log('Document processed:', data);
            console.log('Transactions:', data.transactions);
            console.log('Recommendations:', data.recommendations);
            
            // Store processed data
            processedDocuments[documentId] = {
                transactions: data.transactions || [],
                recommendations: data.recommendations || [],
                processedAt: new Date().toISOString()
            };
            
            console.log('Stored processed data:', processedDocuments[documentId]);
            
            // Reload documents to show updated status
            await loadDocuments();
            
            // Show success message
            showNotification('Документ успешно обработан!', 'success');
        } else {
            const errorData = await response.json();
            showNotification(errorData.error || 'Ошибка обработки документа', 'error');
            if (button) {
                button.disabled = false;
                button.textContent = 'Обработать';
            }
        }
    } catch (error) {
        console.error('Processing error:', error);
        showNotification('Ошибка подключения к серверу', 'error');
        if (button) {
            button.disabled = false;
            button.textContent = 'Обработать';
        }
    }
}

// Load documents
async function loadDocuments() {
    const documentsList = document.getElementById('documentsList');
    documentsList.innerHTML = '<p class="loading">Загрузка...</p>';
    
    try {
        const response = await apiCall('/api/v1/documents?limit=50');
        
        if (response.ok) {
            const documents = await response.json();
            
            if (documents.length === 0) {
                documentsList.innerHTML = `
                    <div class="empty-state">
                        <p>У вас пока нет документов</p>
                        <p>Загрузите первый документ выше</p>
                    </div>
                `;
            } else {
                documentsList.innerHTML = documents.map(doc => {
                    const hasText = doc.extracted_text && doc.extracted_text.length > 0;
                    const status = hasText ? 'processed' : 'uploaded';
                    const processedData = processedDocuments[doc.id];
                    const txCount = processedData?.transactions?.length || 0;
                    const recCount = processedData?.recommendations?.length || 0;
                    
                    // Check if document has been processed (has text or has processed data)
                    const hasProcessedData = hasText || (processedData && (txCount > 0 || recCount > 0));
                    
                    return `
                    <div class="document-card" data-doc-id="${doc.id}">
                        <h3>${escapeHtml(getDocTypeName(doc.type))}</h3>
                        <p><strong>Файл:</strong> ${escapeHtml(doc.file_name)}</p>
                        <p><strong>Размер:</strong> ${formatFileSize(doc.file_size)}</p>
                        <p><strong>Дата:</strong> ${formatDate(doc.created_at)}</p>
                        <span class="status ${status}">${getStatusText(status)}</span>
                        ${txCount > 0 ? `<p><strong>Транзакций:</strong> ${txCount}</p>` : ''}
                        ${recCount > 0 ? `<p><strong>Рекомендаций:</strong> ${recCount}</p>` : ''}
                        ${!hasText ? `
                            <div class="document-actions">
                                <button class="btn btn-primary" onclick="processDocument('${doc.id}', event)">
                                    Обработать
                                </button>
                            </div>
                        ` : ''}
                        ${hasProcessedData ? `
                            <div class="document-actions" style="margin-top: 1rem; display: flex; gap: 0.5rem;">
                                <button class="btn btn-secondary" onclick="showDocumentDetails('${doc.id}')">
                                    Подробнее
                                </button>
                                ${recCount > 0 ? `
                                    <button class="btn btn-primary" onclick="showRecommendations('${doc.id}')">
                                        💡 Рекомендации (${recCount})
                                    </button>
                                ` : ''}
                            </div>
                        ` : ''}
                        ${doc.extracted_text ? `
                            <details style="margin-top: 1rem;">
                                <summary style="cursor: pointer; color: var(--primary-color);">Показать извлеченный текст</summary>
                                <pre style="margin-top: 0.5rem; padding: 0.5rem; background: var(--bg-color); border-radius: 4px; font-size: 0.875rem; max-height: 200px; overflow: auto;">${escapeHtml(doc.extracted_text.substring(0, 500))}${doc.extracted_text.length > 500 ? '...' : ''}</pre>
                            </details>
                        ` : ''}
                    </div>
                `;
                }).join('');
            }
        } else {
            documentsList.innerHTML = '<p class="error-message">Ошибка загрузки документов</p>';
        }
    } catch (error) {
        documentsList.innerHTML = '<p class="error-message">Ошибка подключения к серверу</p>';
    }
}

// Helper functions
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
}

function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleString('ru-RU');
}

function getStatusText(status) {
    const statusMap = {
        'uploaded': 'Загружен',
        'processing': 'Обрабатывается',
        'processed': 'Обработан',
    };
    return statusMap[status] || 'Неизвестно';
}

function getDocTypeName(type) {
    const typeMap = {
        'receipt': 'Чек',
        'statement': 'Выписка',
        'screenshot': 'Скриншот',
    };
    return typeMap[type] || type;
}

// Show document details modal
async function showDocumentDetails(documentId) {
    console.log('Showing details for document:', documentId);
    console.log('Available processed documents:', Object.keys(processedDocuments));
    
    let processedData = processedDocuments[documentId];
    
    // If data not in memory, try to fetch it by processing the document again
    // (this will return cached data if already processed)
    if (!processedData) {
        // Show loading state
        const loadingModal = document.createElement('div');
        loadingModal.innerHTML = `
            <div class="modal-overlay">
                <div class="modal-content">
                    <div class="modal-body">
                        <p class="loading">Загрузка данных...</p>
                    </div>
                </div>
            </div>
        `;
        loadingModal.id = 'loadingModal';
        document.body.appendChild(loadingModal);
        
        try {
            // Try to get data by processing (will return existing data if already processed)
            const response = await apiCall(`/api/v1/documents/${documentId}/process`, {
                method: 'POST',
            });
            
            if (response.ok) {
                const data = await response.json();
                processedData = {
                    transactions: data.transactions || [],
                    recommendations: data.recommendations || [],
                    processedAt: new Date().toISOString()
                };
                processedDocuments[documentId] = processedData;
            }
        } catch (error) {
            console.error('Error fetching document data:', error);
        } finally {
            // Remove loading modal
            const loadingModalEl = document.getElementById('loadingModal');
            if (loadingModalEl) {
                loadingModalEl.remove();
            }
        }
    }
    
    if (!processedData) {
        showNotification('Данные о документе не найдены. Попробуйте обработать документ снова.', 'error');
        return;
    }
    
    const transactions = processedData.transactions || [];
    const recommendations = processedData.recommendations || [];
    
    console.log('Transactions count:', transactions.length);
    console.log('Recommendations count:', recommendations.length);
    console.log('Recommendations data:', recommendations);
    
    let modalContent = `
        <div class="modal-overlay" onclick="closeModal()">
            <div class="modal-content" onclick="event.stopPropagation()">
                <div class="modal-header">
                    <h2>Детали документа</h2>
                    <button class="modal-close" onclick="closeModal()">&times;</button>
                </div>
                <div class="modal-body">
    `;
    
    if (transactions.length > 0) {
        modalContent += `
            <h3>Транзакции (${transactions.length})</h3>
            <div class="transactions-list">
                ${transactions.map(tx => `
                    <div class="transaction-item">
                        <div class="transaction-header">
                            <span class="transaction-amount">${tx.amount} ${tx.currency}</span>
                            <span class="transaction-category">${escapeHtml(tx.category)}</span>
                        </div>
                        <p class="transaction-description">${escapeHtml(tx.description)}</p>
                        ${tx.llm_description ? `<p class="transaction-llm-desc">${escapeHtml(tx.llm_description)}</p>` : ''}
                        <p class="transaction-date">${formatDate(tx.date)}</p>
                    </div>
                `).join('')}
            </div>
        `;
    }
    
    if (recommendations.length > 0) {
        modalContent += `
            <h3 style="margin-top: 2rem;">Рекомендации (${recommendations.length})</h3>
            <div class="recommendations-list">
                ${recommendations.map((rec, index) => {
                    const title = rec.title || `Рекомендация ${index + 1}`;
                    const description = rec.description || '';
                    const savings = rec.potential_savings || rec.potentialSavings || 0;
                    const source = rec.source || 'llm';
                    
                    return `
                    <div class="recommendation-item">
                        <h4>${escapeHtml(title)}</h4>
                        <p>${escapeHtml(description)}</p>
                        ${savings > 0 ? `
                            <p class="recommendation-savings">
                                💰 Потенциальная экономия: ${savings.toFixed(2)} руб
                            </p>
                        ` : ''}
                        <span class="recommendation-source">Источник: ${escapeHtml(source)}</span>
                    </div>
                `;
                }).join('')}
            </div>
        `;
    } else {
        modalContent += `
            <div style="margin-top: 2rem; padding: 1rem; background: var(--bg-color); border-radius: 8px;">
                <p style="color: var(--text-secondary);">Рекомендации не найдены</p>
            </div>
        `;
    }
    
    if (transactions.length === 0 && recommendations.length === 0) {
        modalContent += '<p>Нет данных для отображения</p>';
    }
    
    modalContent += `
                </div>
            </div>
        </div>
    `;
    
    // Create and show modal
    const modal = document.createElement('div');
    modal.innerHTML = modalContent;
    modal.id = 'documentModal';
    document.body.appendChild(modal);
    
    // Log final modal content for debugging
    console.log('Modal content created, recommendations section:', recommendations.length > 0 ? 'present' : 'missing');
}

// Show recommendations modal
async function showRecommendations(documentId) {
    console.log('Showing recommendations for document:', documentId);
    
    let processedData = processedDocuments[documentId];
    
    // If data not in memory, try to fetch it
    if (!processedData) {
        // Show loading state
        const loadingModal = document.createElement('div');
        loadingModal.innerHTML = `
            <div class="modal-overlay">
                <div class="modal-content">
                    <div class="modal-body">
                        <p class="loading">Загрузка рекомендаций...</p>
                    </div>
                </div>
            </div>
        `;
        loadingModal.id = 'loadingModal';
        document.body.appendChild(loadingModal);
        
        try {
            const response = await apiCall(`/api/v1/documents/${documentId}/process`, {
                method: 'POST',
            });
            
            if (response.ok) {
                const data = await response.json();
                processedData = {
                    transactions: data.transactions || [],
                    recommendations: data.recommendations || [],
                    processedAt: new Date().toISOString()
                };
                processedDocuments[documentId] = processedData;
            }
        } catch (error) {
            console.error('Error fetching recommendations:', error);
            showNotification('Ошибка загрузки рекомендаций', 'error');
            const loadingModalEl = document.getElementById('loadingModal');
            if (loadingModalEl) {
                loadingModalEl.remove();
            }
            return;
        } finally {
            const loadingModalEl = document.getElementById('loadingModal');
            if (loadingModalEl) {
                loadingModalEl.remove();
            }
        }
    }
    
    if (!processedData) {
        showNotification('Данные о документе не найдены', 'error');
        return;
    }
    
    const recommendations = processedData.recommendations || [];
    
    if (recommendations.length === 0) {
        showNotification('Рекомендации не найдены', 'info');
        return;
    }
    
    // Calculate total potential savings
    const totalSavings = recommendations.reduce((sum, rec) => {
        const savings = rec.potential_savings || rec.potentialSavings || 0;
        return sum + savings;
    }, 0);
    
    let modalContent = `
        <div class="modal-overlay" onclick="closeModal()">
            <div class="modal-content modal-recommendations" onclick="event.stopPropagation()">
                <div class="modal-header">
                    <h2>💡 Рекомендации по оптимизации расходов</h2>
                    <button class="modal-close" onclick="closeModal()">&times;</button>
                </div>
                <div class="modal-body">
                    ${totalSavings > 0 ? `
                        <div class="total-savings-banner">
                            <h3>💰 Потенциальная экономия: ${totalSavings.toFixed(2)} руб</h3>
                        </div>
                    ` : ''}
                    <div class="recommendations-list">
                        ${recommendations.map((rec, index) => {
                            const title = rec.title || `Рекомендация ${index + 1}`;
                            const description = rec.description || '';
                            const savings = rec.potential_savings || rec.potentialSavings || 0;
                            const source = rec.source || 'llm';
                            const sourceName = getSourceName(source);
                            
                            return `
                            <div class="recommendation-item">
                                <div class="recommendation-number">${index + 1}</div>
                                <div class="recommendation-content">
                                    <h4>${escapeHtml(title)}</h4>
                                    <p class="recommendation-description">${escapeHtml(description)}</p>
                                    ${savings > 0 ? `
                                        <div class="recommendation-savings">
                                            <span class="savings-icon">💰</span>
                                            <span class="savings-amount">Экономия: ${savings.toFixed(2)} руб</span>
                                        </div>
                                    ` : ''}
                                    <div class="recommendation-footer">
                                        <span class="recommendation-source">
                                            <span class="source-icon">📚</span>
                                            ${escapeHtml(sourceName)}
                                        </span>
                                    </div>
                                </div>
                            </div>
                        `;
                        }).join('')}
                    </div>
                </div>
            </div>
        </div>
    `;
    
    // Create and show modal
    const modal = document.createElement('div');
    modal.innerHTML = modalContent;
    modal.id = 'recommendationsModal';
    document.body.appendChild(modal);
}

// Get source name in Russian
function getSourceName(source) {
    const sourceMap = {
        'bank_tariff': 'Тарифы банков',
        'gov_tariff': 'Государственные тарифы',
        'education': 'Финансовая грамотность',
        'llm': 'Анализ ИИ'
    };
    return sourceMap[source] || source;
}

// Close modal
function closeModal() {
    const documentModal = document.getElementById('documentModal');
    if (documentModal) {
        documentModal.remove();
    }
    const recommendationsModal = document.getElementById('recommendationsModal');
    if (recommendationsModal) {
        recommendationsModal.remove();
    }
}

// Show notification
function showNotification(message, type = 'info') {
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.textContent = message;
    document.body.appendChild(notification);
    
    setTimeout(() => {
        notification.classList.add('show');
    }, 10);
    
    setTimeout(() => {
        notification.classList.remove('show');
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    checkAuth();
    
    // Set default values
    document.getElementById('email').value = 'nikita2@gmail.com';
    document.getElementById('password').value = '128912';
});

