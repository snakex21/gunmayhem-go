function lockup()
{
   islock = true;
   this.useHandCursor = false;
   frame.gotoAndStop(1);
}
function lockup2()
{
   islock = false;
   this.useHandCursor = true;
   frame.gotoAndStop(1);
}
function taketheinput()
{
   _root.savedata2.data.controlarray[_parent.number][number] = _parent._parent.getinput;
   _parent.refreshkeys();
}
_parent._parent.btnarray[_parent._parent.btnarray.length] = this;
islock = true;
this.useHandCursor = false;
if(keytext.text == "Up Arrow")
{
   gotoAndStop(3);
}
if(keytext.text == "Left Arrow")
{
   gotoAndStop(4);
}
if(keytext.text == "Down Arrow")
{
   gotoAndStop(5);
}
if(keytext.text == "Right Arrow")
{
   gotoAndStop(6);
}
switch(this._name)
{
   case "key1":
      number = 0;
      break;
   case "key2":
      number = 1;
      break;
   case "key3":
      number = 2;
      break;
   case "key4":
      number = 3;
      break;
   case "key5":
      number = 4;
      break;
   case "key6":
      number = 5;
}
this.onRollOver = function()
{
   if(!islock)
   {
      frame.gotoAndStop(2);
   }
};
this.onRollOut = function()
{
   if(!islock)
   {
      frame.gotoAndStop(1);
   }
};
this.onPress = function()
{
   if(!islock)
   {
      _parent._parent.targetkey = this;
      frame.gotoAndStop(3);
      _parent._parent.disableall();
      _parent._parent.lockup._alpha = 100;
   }
};
