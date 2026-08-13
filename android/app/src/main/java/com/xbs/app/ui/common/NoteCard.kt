package com.xbs.app.ui.common

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.FavoriteBorder
import androidx.compose.material3.Card
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import com.xbs.app.data.api.dto.NoteDto

@Composable
fun NoteCard(note: NoteDto, aspectRatio: Float, onClick: () -> Unit) {
    Card(onClick = onClick, modifier = Modifier.padding(4.dp)) {
        Column {
            AsyncImage(
                model = note.coverUrl,
                contentDescription = note.title,
                contentScale = ContentScale.Crop,
                modifier = Modifier.fillMaxWidth().aspectRatio(aspectRatio),
            )
            Column(Modifier.padding(8.dp)) {
                Text(note.title, maxLines = 2, overflow = TextOverflow.Ellipsis, style = MaterialTheme.typography.bodyMedium)
                Row(
                    Modifier.fillMaxWidth().padding(top = 6.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Spacer(Modifier.weight(1f))
                    Icon(Icons.Default.FavoriteBorder, contentDescription = "点赞", modifier = Modifier.size(14.dp))
                    Spacer(Modifier.width(2.dp))
                    Text("${note.likeCount}", style = MaterialTheme.typography.labelSmall)
                }
            }
        }
    }
}
