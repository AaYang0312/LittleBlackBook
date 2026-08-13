package com.xbs.app.data.repository

import com.xbs.app.data.api.FeedApi
import com.xbs.app.data.api.dataOrThrow
import com.xbs.app.data.api.dto.NoteDto
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class FeedRepository @Inject constructor(
    private val api: FeedApi,
) {
    suspend fun following(offset: Int, size: Int = 20): Result<List<NoteDto>> = runCatching {
        api.following(offset, size).dataOrThrow()
    }
}
