from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.start_pty_options import StartPTYOptions


T = TypeVar("T", bound="SplitPaneRequest")


@_attrs_define
class SplitPaneRequest:
    """
    Attributes:
        direction (str):
        session_id (str):
        target_pane_id (str):
        window_id (str):
        initial_pty (None | StartPTYOptions | Unset):
    """

    direction: str
    session_id: str
    target_pane_id: str
    window_id: str
    initial_pty: None | StartPTYOptions | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.start_pty_options import StartPTYOptions

        direction = self.direction

        session_id = self.session_id

        target_pane_id = self.target_pane_id

        window_id = self.window_id

        initial_pty: dict[str, Any] | None | Unset
        if isinstance(self.initial_pty, Unset):
            initial_pty = UNSET
        elif isinstance(self.initial_pty, StartPTYOptions):
            initial_pty = self.initial_pty.to_dict()
        else:
            initial_pty = self.initial_pty

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "direction": direction,
                "sessionId": session_id,
                "targetPaneId": target_pane_id,
                "windowId": window_id,
            }
        )
        if initial_pty is not UNSET:
            field_dict["initialPty"] = initial_pty

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.start_pty_options import StartPTYOptions

        d = dict(src_dict)
        direction = d.pop("direction")

        session_id = d.pop("sessionId")

        target_pane_id = d.pop("targetPaneId")

        window_id = d.pop("windowId")

        def _parse_initial_pty(data: object) -> None | StartPTYOptions | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                initial_pty_type_1 = StartPTYOptions.from_dict(data)

                return initial_pty_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | StartPTYOptions | Unset, data)

        initial_pty = _parse_initial_pty(d.pop("initialPty", UNSET))

        split_pane_request = cls(
            direction=direction,
            session_id=session_id,
            target_pane_id=target_pane_id,
            window_id=window_id,
            initial_pty=initial_pty,
        )

        split_pane_request.additional_properties = d
        return split_pane_request

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
