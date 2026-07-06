from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.terminal_snapshot import TerminalSnapshot


T = TypeVar("T", bound="OutputSnapshot")


@_attrs_define
class OutputSnapshot:
    """
    Attributes:
        offset (int):
        output (str):
        output_base_64 (str):
        pty_id (str):
        terminal_snapshot (None | TerminalSnapshot | Unset):
    """

    offset: int
    output: str
    output_base_64: str
    pty_id: str
    terminal_snapshot: None | TerminalSnapshot | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.terminal_snapshot import TerminalSnapshot

        offset = self.offset

        output = self.output

        output_base_64 = self.output_base_64

        pty_id = self.pty_id

        terminal_snapshot: dict[str, Any] | None | Unset
        if isinstance(self.terminal_snapshot, Unset):
            terminal_snapshot = UNSET
        elif isinstance(self.terminal_snapshot, TerminalSnapshot):
            terminal_snapshot = self.terminal_snapshot.to_dict()
        else:
            terminal_snapshot = self.terminal_snapshot

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "offset": offset,
                "output": output,
                "outputBase64": output_base_64,
                "ptyId": pty_id,
            }
        )
        if terminal_snapshot is not UNSET:
            field_dict["terminalSnapshot"] = terminal_snapshot

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.terminal_snapshot import TerminalSnapshot

        d = dict(src_dict)
        offset = d.pop("offset")

        output = d.pop("output")

        output_base_64 = d.pop("outputBase64")

        pty_id = d.pop("ptyId")

        def _parse_terminal_snapshot(data: object) -> None | TerminalSnapshot | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                terminal_snapshot_type_1 = TerminalSnapshot.from_dict(data)

                return terminal_snapshot_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | TerminalSnapshot | Unset, data)

        terminal_snapshot = _parse_terminal_snapshot(d.pop("terminalSnapshot", UNSET))

        output_snapshot = cls(
            offset=offset,
            output=output,
            output_base_64=output_base_64,
            pty_id=pty_id,
            terminal_snapshot=terminal_snapshot,
        )

        output_snapshot.additional_properties = d
        return output_snapshot

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
