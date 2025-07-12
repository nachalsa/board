// 전역 변수
let selectedFiles = [];
let uploadInProgress = false;

// DOM이 로드되면 초기화
document.addEventListener('DOMContentLoaded', function() {
    initializeDragAndDrop();
    initializeFileInput();
    initializeForms();
    initializeContentToggle();
});

// 드래그 앤 드롭 초기화
function initializeDragAndDrop() {
    const fileUploadArea = document.getElementById('fileUploadArea');
    
    // 드래그 이벤트 방지 (전체 페이지)
    ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
        document.addEventListener(eventName, preventDefaults, false);
    });
    
    function preventDefaults(e) {
        e.preventDefault();
        e.stopPropagation();
    }
    
    // 드래그 앤 드롭 영역 이벤트
    ['dragenter', 'dragover'].forEach(eventName => {
        fileUploadArea.addEventListener(eventName, handleDragEnter, false);
    });
    
    ['dragleave', 'drop'].forEach(eventName => {
        fileUploadArea.addEventListener(eventName, handleDragLeave, false);
    });
    
    fileUploadArea.addEventListener('drop', handleFileDrop, false);
    fileUploadArea.addEventListener('click', () => {
        document.getElementById('fileInput').click();
    });
    
    function handleDragEnter(e) {
        fileUploadArea.classList.add('dragover');
    }
    
    function handleDragLeave(e) {
        // 실제로 영역을 벗어났는지 확인
        if (!fileUploadArea.contains(e.relatedTarget)) {
            fileUploadArea.classList.remove('dragover');
        }
    }
    
    function handleFileDrop(e) {
        fileUploadArea.classList.remove('dragover');
        const files = Array.from(e.dataTransfer.files);
        handleFileSelection(files);
    }
}

// 파일 입력 초기화
function initializeFileInput() {
    const fileInput = document.getElementById('fileInput');
    
    fileInput.addEventListener('change', function(e) {
        const files = Array.from(e.target.files);
        handleFileSelection(files);
    });
}

// 파일 선택 처리
function handleFileSelection(files) {
    if (uploadInProgress) {
        showNotification('업로드가 진행 중입니다. 잠시 후 다시 시도해주세요.', 'info');
        return;
    }
    
    // 파일 크기 검증 (500MB = 500 * 1024 * 1024)
    const maxSize = 500 * 1024 * 1024;
    const validFiles = [];
    const invalidFiles = [];
    
    files.forEach(file => {
        if (file.size > maxSize) {
            invalidFiles.push(file);
        } else {
            validFiles.push(file);
        }
    });
    
    // 크기 초과 파일 알림
    if (invalidFiles.length > 0) {
        const fileNames = invalidFiles.map(f => f.name).join(', ');
        showNotification(`다음 파일들이 500MB를 초과합니다: ${fileNames}`, 'error');
    }
    
    // 유효한 파일들 추가
    if (validFiles.length > 0) {
        selectedFiles = [...selectedFiles, ...validFiles];
        updateFileList();
        updateUploadButton();
    }
}

// 파일 목록 업데이트
function updateFileList() {
    const fileList = document.getElementById('fileList');
    
    if (selectedFiles.length === 0) {
        fileList.innerHTML = '';
        return;
    }
    
    fileList.innerHTML = selectedFiles.map((file, index) => `
        <div class="file-item">
            <div class="file-info">
                <div class="file-name">${file.name}</div>
                <div class="file-size">${formatFileSize(file.size)}</div>
            </div>
            <button class="remove-file" onclick="removeFile(${index})">제거</button>
        </div>
    `).join('');
}

// 파일 크기 포맷팅
function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// 파일 제거
function removeFile(index) {
    selectedFiles.splice(index, 1);
    updateFileList();
    updateUploadButton();
}

// 업로드 버튼 상태 업데이트
function updateUploadButton() {
    const uploadBtn = document.querySelector('#fileUploadForm .upload-btn');
    uploadBtn.disabled = selectedFiles.length === 0 || uploadInProgress;
}

// 파일 업로드 초기화
function resetFileUpload() {
    selectedFiles = [];
    document.getElementById('fileInput').value = '';
    document.getElementById('fileTitle').value = '';
    updateFileList();
    updateUploadButton();
    document.getElementById('fileUploadArea').classList.remove('dragover');
}

// 폼 초기화
function initializeForms() {
    // 파일 업로드 폼
    const fileUploadForm = document.getElementById('fileUploadForm');
    fileUploadForm.addEventListener('submit', handleFileUpload);
    
    // 메시지 업로드 폼
    const messageUploadForm = document.getElementById('messageUploadForm');
    messageUploadForm.addEventListener('submit', handleMessageUpload);
}

// 파일 업로드 처리
async function handleFileUpload(e) {
    e.preventDefault();
    
    if (selectedFiles.length === 0) {
        showNotification('업로드할 파일을 선택해주세요.', 'error');
        return;
    }
    
    if (uploadInProgress) {
        showNotification('업로드가 이미 진행 중입니다.', 'info');
        return;
    }
    
    uploadInProgress = true;
    showLoadingOverlay(true);
    
    const title = document.getElementById('fileTitle').value;
    let successCount = 0;
    let errorCount = 0;
    
    try {
        // 각 파일을 순차적으로 업로드
        for (const file of selectedFiles) {
            try {
                await uploadSingleFile(file, title);
                successCount++;
            } catch (error) {
                console.error('파일 업로드 실패:', error);
                errorCount++;
            }
        }
        
        // 결과 알림
        if (successCount > 0 && errorCount === 0) {
            showNotification(`${successCount}개 파일이 성공적으로 업로드되었습니다.`, 'success');
            resetFileUpload();
            setTimeout(() => location.reload(), 1500);
        } else if (successCount > 0 && errorCount > 0) {
            showNotification(`${successCount}개 성공, ${errorCount}개 실패`, 'info');
        } else {
            showNotification('모든 파일 업로드에 실패했습니다.', 'error');
        }
        
    } catch (error) {
        console.error('업로드 중 오류:', error);
        showNotification('업로드 중 오류가 발생했습니다.', 'error');
    } finally {
        uploadInProgress = false;
        showLoadingOverlay(false);
        updateUploadButton();
    }
}

// 단일 파일 업로드
function uploadSingleFile(file, title) {
    return new Promise((resolve, reject) => {
        const formData = new FormData();
        formData.append('file', file);
        formData.append('title', title || file.name);
        
        fetch('/upload/file', {
            method: 'POST',
            body: formData
        })
        .then(response => response.json())
        .then(data => {
            if (data.message) {
                resolve(data);
            } else {
                reject(new Error(data.error || '업로드 실패'));
            }
        })
        .catch(error => {
            reject(error);
        });
    });
}

// 메시지 업로드 처리
function handleMessageUpload(e) {
    e.preventDefault();
    
    if (uploadInProgress) {
        showNotification('업로드가 진행 중입니다. 잠시 후 다시 시도해주세요.', 'info');
        return;
    }
    
    const title = document.getElementById('messageTitle').value.trim();
    const content = document.getElementById('messageContent').value.trim();
    
    if (!title) {
        showNotification('제목을 입력해주세요.', 'error');
        return;
    }
    
    if (!content) {
        showNotification('내용을 입력해주세요.', 'error');
        return;
    }
    
    uploadInProgress = true;
    showLoadingOverlay(true);
    
    const formData = new FormData();
    formData.append('title', title);
    formData.append('content', content);
    
    fetch('/upload/message', {
        method: 'POST',
        body: formData
    })
    .then(response => response.json())
    .then(data => {
        if (data.message) {
            showNotification('메시지가 성공적으로 업로드되었습니다.', 'success');
            document.getElementById('messageUploadForm').reset();
            setTimeout(() => location.reload(), 1500);
        } else {
            showNotification(data.error || '메시지 업로드에 실패했습니다.', 'error');
        }
    })
    .catch(error => {
        console.error('메시지 업로드 실패:', error);
        showNotification('메시지 업로드 중 오류가 발생했습니다.', 'error');
    })
    .finally(() => {
        uploadInProgress = false;
        showLoadingOverlay(false);
    });
}

// 내용 토글 초기화
function initializeContentToggle() {
    // 페이지 로드시 모든 내용 숨기기
    document.querySelectorAll('.post-content').forEach(content => {
        content.style.display = 'none';
    });
}

// 게시글 내용 토글
function toggleContent(postId) {
    const content = document.getElementById(`content-${postId}`);
    const icon = document.getElementById(`toggle-icon-${postId}`);
    
    if (!content) return;
    
    if (content.style.display === 'none' || content.style.display === '') {
        // 내용 표시
        content.style.display = 'block';
        icon.textContent = '▲';
        
        // 스크롤 애니메이션
        setTimeout(() => {
            content.scrollIntoView({ 
                behavior: 'smooth', 
                block: 'nearest' 
            });
        }, 100);
    } else {
        // 내용 숨기기
        content.style.display = 'none';
        icon.textContent = '▼';
    }
}

// 로딩 오버레이 표시/숨기기
function showLoadingOverlay(show) {
    const overlay = document.getElementById('loadingOverlay');
    overlay.style.display = show ? 'flex' : 'none';
}

// 알림 메시지 표시
function showNotification(message, type = 'info') {
    const notification = document.getElementById('notification');
    
    notification.textContent = message;
    notification.className = `notification ${type}`;
    
    // 알림 표시
    setTimeout(() => {
        notification.classList.add('show');
    }, 100);
    
    // 3초 후 자동 숨김
    setTimeout(() => {
        notification.classList.remove('show');
    }, 3000);
}

// 파일 다운로드 처리 (분석용)
function trackDownload(postId, fileName) {
    console.log(`파일 다운로드: ID ${postId}, 파일명: ${fileName}`);
    // 여기에 다운로드 통계 등을 추가할 수 있습니다.
}

// 에러 핸들링
window.addEventListener('error', function(e) {
    console.error('JavaScript 에러:', e.error);
    showNotification('페이지에 오류가 발생했습니다.', 'error');
});

// 네트워크 에러 핸들링
window.addEventListener('unhandledrejection', function(e) {
    console.error('Promise 에러:', e.reason);
    showNotification('네트워크 오류가 발생했습니다.', 'error');
});

// 페이지 언로드시 업로드 중단 경고
window.addEventListener('beforeunload', function(e) {
    if (uploadInProgress) {
        e.preventDefault();
        e.returnValue = '업로드가 진행 중입니다. 페이지를 나가시겠습니까?';
        return e.returnValue;
    }
});

// 유틸리티 함수들
const utils = {
    // 파일 확장자 검사
    isValidFileType: function(fileName, allowedTypes) {
        const extension = fileName.split('.').pop().toLowerCase();
        return allowedTypes.includes(extension);
    },
    
    // 텍스트 길이 제한
    truncateText: function(text, maxLength) {
        if (text.length <= maxLength) return text;
        return text.substring(0, maxLength) + '...';
    },
    
    // 시간 포맷팅
    formatDate: function(dateString) {
        const date = new Date(dateString);
        return date.toLocaleDateString('ko-KR', {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    },
    
    // 디바운스 함수
    debounce: function(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    }
};

// 개발 환경에서 디버깅 정보
if (window.location.hostname === 'localhost') {
    console.log('파일 업로드 게시판 - 개발 모드');
    console.log('사용 가능한 함수:', {
        toggleContent: 'toggleContent(postId)',
        resetFileUpload: 'resetFileUpload()',
        showNotification: 'showNotification(message, type)',
        utils: 'utils 객체의 유틸리티 함수들'
    });
}