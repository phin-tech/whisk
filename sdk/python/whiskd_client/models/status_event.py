from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="StatusEvent")


@_attrs_define
class StatusEvent:
    """
    Attributes:
        created_at (datetime.datetime):
        id (str):
        kind (str):
        message (str):
        requires_attention (bool):
        scope (str):
        actor (str | Unset):
        project_id (str | Unset):
        pty_id (str | Unset):
        read_at (datetime.datetime | None | Unset):
        run_id (str | Unset):
        session_id (str | Unset):
        work_item_id (str | Unset):
    """

    created_at: datetime.datetime
    id: str
    kind: str
    message: str
    requires_attention: bool
    scope: str
    actor: str | Unset = UNSET
    project_id: str | Unset = UNSET
    pty_id: str | Unset = UNSET
    read_at: datetime.datetime | None | Unset = UNSET
    run_id: str | Unset = UNSET
    session_id: str | Unset = UNSET
    work_item_id: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        id = self.id

        kind = self.kind

        message = self.message

        requires_attention = self.requires_attention

        scope = self.scope

        actor = self.actor

        project_id = self.project_id

        pty_id = self.pty_id

        read_at: None | str | Unset
        if isinstance(self.read_at, Unset):
            read_at = UNSET
        elif isinstance(self.read_at, datetime.datetime):
            read_at = self.read_at.isoformat()
        else:
            read_at = self.read_at

        run_id = self.run_id

        session_id = self.session_id

        work_item_id = self.work_item_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "id": id,
                "kind": kind,
                "message": message,
                "requiresAttention": requires_attention,
                "scope": scope,
            }
        )
        if actor is not UNSET:
            field_dict["actor"] = actor
        if project_id is not UNSET:
            field_dict["projectId"] = project_id
        if pty_id is not UNSET:
            field_dict["ptyId"] = pty_id
        if read_at is not UNSET:
            field_dict["readAt"] = read_at
        if run_id is not UNSET:
            field_dict["runId"] = run_id
        if session_id is not UNSET:
            field_dict["sessionId"] = session_id
        if work_item_id is not UNSET:
            field_dict["workItemId"] = work_item_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        id = d.pop("id")

        kind = d.pop("kind")

        message = d.pop("message")

        requires_attention = d.pop("requiresAttention")

        scope = d.pop("scope")

        actor = d.pop("actor", UNSET)

        project_id = d.pop("projectId", UNSET)

        pty_id = d.pop("ptyId", UNSET)

        def _parse_read_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                read_at_type_0 = datetime.datetime.fromisoformat(data)

                return read_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        read_at = _parse_read_at(d.pop("readAt", UNSET))

        run_id = d.pop("runId", UNSET)

        session_id = d.pop("sessionId", UNSET)

        work_item_id = d.pop("workItemId", UNSET)

        status_event = cls(
            created_at=created_at,
            id=id,
            kind=kind,
            message=message,
            requires_attention=requires_attention,
            scope=scope,
            actor=actor,
            project_id=project_id,
            pty_id=pty_id,
            read_at=read_at,
            run_id=run_id,
            session_id=session_id,
            work_item_id=work_item_id,
        )

        status_event.additional_properties = d
        return status_event

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
