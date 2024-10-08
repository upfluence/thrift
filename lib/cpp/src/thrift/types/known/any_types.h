/**
 * Autogenerated by Thrift Compiler (2.5.4-upfluence)
 *
 * DO NOT EDIT UNLESS YOU ARE SURE THAT YOU KNOW WHAT YOU ARE DOING
 *  @generated
 */
#ifndef any_TYPES_H
#define any_TYPES_H

#include <iosfwd>

#include <thrift/Thrift.h>
#include <thrift/TApplicationException.h>
#include <thrift/protocol/TProtocol.h>
#include <thrift/transport/TTransport.h>

#include <thrift/cxxfunctional.h>


namespace types { namespace known { namespace any {

class Any;


class Any {
 public:

  static const char* ascii_fingerprint; // = "07A9615F837F7D0A952B595DD3020972";
  static const uint8_t binary_fingerprint[16]; // = {0x07,0xA9,0x61,0x5F,0x83,0x7F,0x7D,0x0A,0x95,0x2B,0x59,0x5D,0xD3,0x02,0x09,0x72};

  Any(const Any&);
  Any& operator=(const Any&);
  Any() : type(), value() {
  }

  virtual ~Any() throw();
  std::string type;
  std::string value;

  void __set_type(const std::string& val);

  void __set_value(const std::string& val);

  bool operator == (const Any & rhs) const
  {
    if (!(type == rhs.type))
      return false;
    if (!(value == rhs.value))
      return false;
    return true;
  }
  bool operator != (const Any &rhs) const {
    return !(*this == rhs);
  }

  bool operator < (const Any & ) const;

  uint32_t read(::apache::thrift::protocol::TProtocol* iprot);
  uint32_t write(::apache::thrift::protocol::TProtocol* oprot) const;

  friend std::ostream& operator<<(std::ostream& out, const Any& obj);
};

void swap(Any &a, Any &b);

}}} // namespace

#endif
