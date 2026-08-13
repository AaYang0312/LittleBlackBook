package com.xbs.app.ui.publish

import android.content.ContentResolver
import android.net.Uri
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.xbs.app.data.repository.NoteRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class PublishUiState(
    val uris: List<Uri> = emptyList(),
    val isPublishing: Boolean = false,
    val uploadedCount: Int = 0,
    val error: String? = null,
)

const val MAX_IMAGES = 9

@HiltViewModel
class PublishViewModel @Inject constructor(
    private val repo: NoteRepository,
) : ViewModel() {

    private val _uiState = MutableStateFlow(PublishUiState())
    val uiState: StateFlow<PublishUiState> = _uiState.asStateFlow()

    private val _published = MutableSharedFlow<Unit>(extraBufferCapacity = 1)
    val published: SharedFlow<Unit> = _published.asSharedFlow()

    fun addImages(newUris: List<Uri>) {
        _uiState.update { it.copy(uris = (it.uris + newUris).take(MAX_IMAGES), error = null) }
    }

    fun removeImage(uri: Uri) {
        _uiState.update { it.copy(uris = it.uris - uri) }
    }

    fun publish(contentResolver: ContentResolver, title: String, content: String) {
        val uris = _uiState.value.uris
        if (title.isBlank()) { _uiState.update { it.copy(error = "标题不能为空") }; return }
        if (uris.isEmpty()) { _uiState.update { it.copy(error = "至少选择一张图片") }; return }
        if (_uiState.value.isPublishing) return

        viewModelScope.launch {
            _uiState.update { it.copy(isPublishing = true, uploadedCount = 0, error = null) }
            // 逐张上传，任一失败即终止
            val urls = mutableListOf<String>()
            for ((i, uri) in uris.withIndex()) {
                val bytes = runCatching { contentResolver.openInputStream(uri)?.use { it.readBytes() } }.getOrNull()
                if (bytes == null) {
                    _uiState.update { it.copy(isPublishing = false, error = "读取图片失败") }
                    return@launch
                }
                val result = repo.uploadImage(bytes, "img_${System.currentTimeMillis()}_$i.jpg")
                result.onSuccess { url ->
                    urls.add(url)
                    _uiState.update { it.copy(uploadedCount = urls.size) }
                }.onFailure { e ->
                    _uiState.update { it.copy(isPublishing = false, error = "图片上传失败：${e.message}") }
                    return@launch
                }
            }
            repo.publish(title.trim(), content.trim(), urls)
                .onSuccess { _published.emit(Unit) }
                .onFailure { e -> _uiState.update { it.copy(error = "发布失败：${e.message}") } }
            _uiState.update { it.copy(isPublishing = false) }
        }
    }
}
