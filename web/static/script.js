// 전역 변수
let selectedFiles = [];
let uploadInProgress = false;
let isReloading = false; // 페이지 새로고침 플래그 추가

// 설정값 가져오기
function getMaxFileSize() {
    return window.APP_CONFIG ? window.APP_CONFIG.maxFileSizeMB * 1024 * 1024 : 500 * 1024 * 1024;
}

function getMaxFileSizeText() {
    return window.APP_CONFIG ? window.APP_CONFIG.maxFileSizeText : '500MB';
}

// DOM이 로드되면 초기화
document.addEventListener('DOMContentLoaded', function() {
    initializeDragAndDrop();
    initializeFileInput();
    initializeForms();
    initializeContentToggle();
    updateAllDates();
    
    setInterval(updateAllDates, 60000); 
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
    
    // 기존 이벤트 리스너 제거 (있다면)
    fileInput.removeEventListener('change', handleFileInputChange);
    
    // 새 이벤트 리스너 추가
    fileInput.addEventListener('change', handleFileInputChange);
}

// 파일 입력 변경 핸들러 (분리하여 참조 가능하게 함)
function handleFileInputChange(e) {
    const files = Array.from(e.target.files);
    if (files.length > 0) {
        handleFileSelection(files);
        // 파일 선택 후 input을 초기화하여 같은 파일을 다시 선택할 수 있게 함
        e.target.value = '';
    }
}

// 파일 선택 처리
function handleFileSelection(files) {
    if (uploadInProgress) {
        showNotification('업로드가 진행 중입니다. 잠시 후 다시 시도해주세요.', 'info');
        return;
    }
    
    // 파일 크기 검증 (동적으로 설정값 사용)
    const maxSize = getMaxFileSize();
    const maxSizeText = getMaxFileSizeText();
    const validFiles = [];
    const invalidFiles = [];
    const duplicateFiles = [];
    
    files.forEach(file => {
        // 중복 파일 체크 (이름과 크기로 판단)
        const isDuplicate = selectedFiles.some(existingFile => 
            existingFile.name === file.name && existingFile.size === file.size
        );
        
        if (isDuplicate) {
            duplicateFiles.push(file);
        } else if (file.size > maxSize) {
            invalidFiles.push(file);
        } else {
            validFiles.push(file);
        }
    });
    
    // 알림 메시지
    if (invalidFiles.length > 0) {
        const fileNames = invalidFiles.map(f => f.name).join(', ');
        showNotification(`다음 파일들이 ${maxSizeText}를 초과합니다: ${fileNames}`, 'error');
    }
    
    if (duplicateFiles.length > 0) {
        const fileNames = duplicateFiles.map(f => f.name).join(', ');
        showNotification(`다음 파일들은 이미 선택되었습니다: ${fileNames}`, 'warning');
    }
    
    // 유효한 파일들 추가
    if (validFiles.length > 0) {
        selectedFiles = [...selectedFiles, ...validFiles];
        updateFileList();
        updateUploadButton();
        showNotification(`${validFiles.length}개 파일이 선택되었습니다.`, 'success');
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
    const fileInput = document.getElementById('fileInput');
    fileInput.value = '';
    // 파일 입력을 완전히 초기화하기 위해 복제 후 교체
    const newFileInput = fileInput.cloneNode(true);
    fileInput.parentNode.replaceChild(newFileInput, fileInput);
    // 새로운 입력에 이벤트 리스너 다시 연결
    initializeFileInput();
    
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
            showNotification(`${successCount}개 파일이 업로드되었습니다.`, 'success');
            resetFileUpload();
            isReloading = true; // 새로고침 플래그 설정
            location.reload();
        } else if (successCount > 0 && errorCount > 0) {
            showNotification(`${successCount}개 성공, ${errorCount}개 실패`, 'info');
            resetFileUpload();
        } else {
            showNotification('파일 업로드에 실패했습니다.', 'error');
        }
        
    } catch (error) {
        console.error('업로드 중 오류:', error);
        showNotification('업로드 중 오류가 발생했습니다.', 'error');
        resetFileUpload(); // 오류 발생시에도 파일 입력 초기화
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
    
    // 내용은 선택사항이므로 빈 값도 허용
    
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
            showNotification('메시지가 업로드되었습니다.', 'success');
            document.getElementById('messageUploadForm').reset();
            isReloading = true; // 새로고침 플래그 설정
            location.reload();
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
    
    if (!content || !icon) return;
    
    if (content.style.display === 'none' || content.style.display === '') {
        content.style.display = 'block';
        icon.textContent = '▲';
    } else {
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

// 페이지 언로드시 업로드 중단 경고
window.addEventListener('beforeunload', function(e) {
    // 의도적인 새로고침이 아니고 업로드가 진행 중일 때만 경고
    if (uploadInProgress && !isReloading) {
        e.preventDefault();
        e.returnValue = '업로드가 진행 중입니다. 페이지를 나가시겠습니까?';
        return e.returnValue;
    }
});

// 시간 'n분 전' 형태로 표시
function timeAgo(dateString) {
    const now = new Date();
    const past = new Date(dateString);
    const seconds = Math.floor((now - past) / 1000);

    const intervals = [
        { label: '년', seconds: 31536000 },
        { label: '개월', seconds: 2592000 },
        { label: '일', seconds: 86400 },
        { label: '시간', seconds: 3600 },
        { label: '분', seconds: 60 }
    ];

    for (let interval of intervals) {
        const count = Math.floor(seconds / interval.seconds);
        if (count > 0) return count + interval.label + ' 전';
    }
    
    return seconds < 10 ? '방금 전' : Math.floor(seconds) + '초 전';
}

// 모든 날짜 업데이트
function updateAllDates() {
    document.querySelectorAll('.post-date[data-timestamp]').forEach(el => {
        const timestamp = el.dataset.timestamp;
        el.textContent = timeAgo(timestamp);
        el.title = new Date(timestamp).toLocaleString('ko-KR');
    });
}

