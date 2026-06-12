from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.attachment import Attachment
    from ..models.history_event import HistoryEvent
    from ..models.worktree_binding import WorktreeBinding


T = TypeVar("T", bound="WorkItem")


@_attrs_define
class WorkItem:
    """
    Attributes:
        attachments (list[Attachment] | None):
        body_markdown (str):
        created_at (datetime.datetime):
        history (list[HistoryEvent] | None):
        id (str):
        number (int):
        project_id (str):
        run_state (str):
        stage_id (str):
        title (str):
        updated_at (datetime.datetime):
        worktree (None | Unset | WorktreeBinding):
    """

    attachments: list[Attachment] | None
    body_markdown: str
    created_at: datetime.datetime
    history: list[HistoryEvent] | None
    id: str
    number: int
    project_id: str
    run_state: str
    stage_id: str
    title: str
    updated_at: datetime.datetime
    worktree: None | Unset | WorktreeBinding = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.worktree_binding import WorktreeBinding

        attachments: list[dict[str, Any]] | None
        if isinstance(self.attachments, list):
            attachments = []
            for attachments_type_0_item_data in self.attachments:
                attachments_type_0_item = attachments_type_0_item_data.to_dict()
                attachments.append(attachments_type_0_item)

        else:
            attachments = self.attachments

        body_markdown = self.body_markdown

        created_at = self.created_at.isoformat()

        history: list[dict[str, Any]] | None
        if isinstance(self.history, list):
            history = []
            for history_type_0_item_data in self.history:
                history_type_0_item = history_type_0_item_data.to_dict()
                history.append(history_type_0_item)

        else:
            history = self.history

        id = self.id

        number = self.number

        project_id = self.project_id

        run_state = self.run_state

        stage_id = self.stage_id

        title = self.title

        updated_at = self.updated_at.isoformat()

        worktree: dict[str, Any] | None | Unset
        if isinstance(self.worktree, Unset):
            worktree = UNSET
        elif isinstance(self.worktree, WorktreeBinding):
            worktree = self.worktree.to_dict()
        else:
            worktree = self.worktree

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "attachments": attachments,
                "bodyMarkdown": body_markdown,
                "createdAt": created_at,
                "history": history,
                "id": id,
                "number": number,
                "projectId": project_id,
                "runState": run_state,
                "stageId": stage_id,
                "title": title,
                "updatedAt": updated_at,
            }
        )
        if worktree is not UNSET:
            field_dict["worktree"] = worktree

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.attachment import Attachment
        from ..models.history_event import HistoryEvent
        from ..models.worktree_binding import WorktreeBinding

        d = dict(src_dict)

        def _parse_attachments(data: object) -> list[Attachment] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                attachments_type_0 = []
                _attachments_type_0 = data
                for attachments_type_0_item_data in _attachments_type_0:
                    attachments_type_0_item = Attachment.from_dict(
                        attachments_type_0_item_data
                    )

                    attachments_type_0.append(attachments_type_0_item)

                return attachments_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[Attachment] | None, data)

        attachments = _parse_attachments(d.pop("attachments"))

        body_markdown = d.pop("bodyMarkdown")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        def _parse_history(data: object) -> list[HistoryEvent] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                history_type_0 = []
                _history_type_0 = data
                for history_type_0_item_data in _history_type_0:
                    history_type_0_item = HistoryEvent.from_dict(
                        history_type_0_item_data
                    )

                    history_type_0.append(history_type_0_item)

                return history_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[HistoryEvent] | None, data)

        history = _parse_history(d.pop("history"))

        id = d.pop("id")

        number = d.pop("number")

        project_id = d.pop("projectId")

        run_state = d.pop("runState")

        stage_id = d.pop("stageId")

        title = d.pop("title")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        def _parse_worktree(data: object) -> None | Unset | WorktreeBinding:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                worktree_type_1 = WorktreeBinding.from_dict(data)

                return worktree_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Unset | WorktreeBinding, data)

        worktree = _parse_worktree(d.pop("worktree", UNSET))

        work_item = cls(
            attachments=attachments,
            body_markdown=body_markdown,
            created_at=created_at,
            history=history,
            id=id,
            number=number,
            project_id=project_id,
            run_state=run_state,
            stage_id=stage_id,
            title=title,
            updated_at=updated_at,
            worktree=worktree,
        )

        work_item.additional_properties = d
        return work_item

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
